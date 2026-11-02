package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/stevelittlefish/lemon-chat/internal/comfyui"
)

const (
	comfySubmitTimeout         = 15 * time.Second
	comfyHistoryRequestTimeout = 5 * time.Second
	comfyDownloadTimeout       = 30 * time.Second
)

// comfyPollInterval is a var so tests can poll a fake ComfyUI quickly.
var comfyPollInterval = time.Second

type comfyImage struct {
	Filename  string `json:"filename"`
	Subfolder string `json:"subfolder"`
	Type      string `json:"type"`
}

func submitToComfyUI(comfyBase string, promptPayload []byte) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), comfySubmitTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "POST", comfyBase+"/prompt", bytes.NewReader(promptPayload))
	if err != nil {
		return "", fmt.Errorf("build prompt request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("could not reach ComfyUI at %q: %w — tell the user the image could not be generated because ComfyUI is unreachable and they should check that it is running", comfyBase, err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ComfyUI returned HTTP %d: %s — tell the user the image could not be generated because ComfyUI returned an error", resp.StatusCode, body)
	}
	var pr struct {
		PromptID string `json:"prompt_id"`
	}
	if err := json.Unmarshal(body, &pr); err != nil || pr.PromptID == "" {
		return "", fmt.Errorf("ComfyUI returned an unexpected response (expected a prompt_id): %s — tell the user image generation failed due to an unexpected ComfyUI response", string(body))
	}
	return pr.PromptID, nil
}

// pollComfyUI waits for promptID to finish and returns the first image from
// outputNode. A ComfyUI execution error is returned as soon as it is reported
// rather than waiting out the timeout.
func pollComfyUI(comfyBase, promptID, outputNode string, timeout time.Duration) (*comfyImage, error) {
	type comfyOutput struct {
		Images []comfyImage `json:"images"`
	}
	type comfyJob struct {
		Outputs map[string]comfyOutput `json:"outputs"`
		Status  struct {
			StatusStr string              `json:"status_str"`
			Completed bool                `json:"completed"`
			Messages  [][]json.RawMessage `json:"messages"`
		} `json:"status"`
	}
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		time.Sleep(comfyPollInterval)
		ctx, cancel := context.WithTimeout(context.Background(), comfyHistoryRequestTimeout)
		req, _ := http.NewRequestWithContext(ctx, "GET", comfyBase+"/history/"+promptID, nil)
		resp, err := http.DefaultClient.Do(req)
		cancel()
		if err != nil {
			continue
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		var history map[string]comfyJob
		if err := json.Unmarshal(body, &history); err != nil {
			continue
		}
		job, ok := history[promptID]
		if !ok {
			continue
		}
		if job.Status.StatusStr == "error" {
			return nil, fmt.Errorf("ComfyUI failed to run the workflow: %s — tell the user image generation failed and pass on this error", comfyExecutionError(job.Status.Messages))
		}
		if out, ok := job.Outputs[outputNode]; ok && len(out.Images) > 0 {
			img := out.Images[0]
			return &img, nil
		}
		if job.Status.Completed {
			return nil, fmt.Errorf("ComfyUI finished the workflow but output node %q produced no image — tell the user image generation failed", outputNode)
		}
	}
	return nil, fmt.Errorf("image generation timed out after %d seconds — tell the user the image was not produced in time and suggest checking that ComfyUI is processing jobs correctly", int(timeout/time.Second))
}

// comfyExecutionError pulls the exception out of a history entry's status
// messages ([["execution_error", {...}], ...]).
func comfyExecutionError(messages [][]json.RawMessage) string {
	for _, m := range messages {
		if len(m) != 2 {
			continue
		}
		var kind string
		if json.Unmarshal(m[0], &kind) != nil || kind != "execution_error" {
			continue
		}
		var e struct {
			NodeID           string `json:"node_id"`
			NodeType         string `json:"node_type"`
			ExceptionMessage string `json:"exception_message"`
		}
		if json.Unmarshal(m[1], &e) == nil {
			return fmt.Sprintf("node %s (%s): %s", e.NodeID, e.NodeType, strings.TrimSpace(e.ExceptionMessage))
		}
	}
	return "unknown execution error"
}

func downloadAndSaveImage(dataDir, comfyBase string, img *comfyImage) (string, error) {
	viewURL := comfyBase + "/view?filename=" + url.QueryEscape(img.Filename) +
		"&subfolder=" + url.QueryEscape(img.Subfolder) +
		"&type=" + url.QueryEscape(img.Type)
	ctx, cancel := context.WithTimeout(context.Background(), comfyDownloadTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", viewURL, nil)
	if err != nil {
		return "", fmt.Errorf("build download request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("image was generated but could not be downloaded from ComfyUI: %w — tell the user the image was produced but could not be retrieved", err)
	}
	imgData, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	relDir := filepath.Join("attachments", randomID())
	absDir := filepath.Join(dataDir, relDir)
	if err := os.MkdirAll(absDir, 0755); err != nil {
		return "", fmt.Errorf("server error: could not create storage directory (%w) — tell the user the image was generated but could not be saved due to a server storage problem", err)
	}
	if err := os.WriteFile(filepath.Join(absDir, "image.png"), imgData, 0644); err != nil {
		return "", fmt.Errorf("server error: could not write image to disk (%w) — tell the user the image was generated but could not be saved due to a server storage problem", err)
	}
	return filepath.Join(relDir, "image.png"), nil
}

// imageResult is what generate_image returns to the model. It embeds
// AttachmentResult so messages.go still recognises it as an attachment.
type imageResult struct {
	AttachmentResult
	WorkflowID     string   `json:"workflow_id"`
	WorkflowHash   string   `json:"workflow_hash"`
	Profile        string   `json:"profile,omitempty"`
	Seed           int64    `json:"seed"`
	AspectRatio    string   `json:"aspect_ratio"`
	NegativePrompt *string  `json:"negative_prompt,omitempty"`
	Warnings       []string `json:"warnings,omitempty"`
}

// makeImageExecutor returns the generate_image executor for a loaded registry.
func makeImageExecutor(comfyURL string, reg *comfyui.Registry) toolExecutor {
	comfyBase := strings.TrimRight(comfyURL, "/")
	return func(argsJSON string, tctx ToolContext) (string, error) {
		var args struct {
			Prompt         string  `json:"prompt"`
			Profile        string  `json:"profile"`
			WorkflowID     string  `json:"workflow_id"`
			AspectRatio    string  `json:"aspect_ratio"`
			NegativePrompt *string `json:"negative_prompt"`
			Seed           int64   `json:"seed"`
			Background     bool    `json:"background"`
			Async          *bool   `json:"async"` // nil = default true
		}
		if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
			return "", fmt.Errorf("invalid args: %w", err)
		}
		job, err := reg.Prepare(comfyui.Request{
			Prompt:         args.Prompt,
			Profile:        args.Profile,
			WorkflowID:     args.WorkflowID,
			AspectRatio:    args.AspectRatio,
			Seed:           args.Seed,
			NegativePrompt: args.NegativePrompt,
		})
		if err != nil {
			return "", err
		}
		w := job.Workflow

		promptPayload, _ := json.Marshal(map[string]any{
			"prompt":    job.Graph,
			"client_id": randomID(),
		})

		useAsync := args.Async == nil || *args.Async

		log.Printf("Generating image workflow=%q profile=%q seed=%d size=%dx%d async=%v conversation_id=%d prompt=%q",
			w.ID, job.Profile, job.Seed, job.Width, job.Height, useAsync, tctx.ConversationID, args.Prompt)

		result := imageResult{
			AttachmentResult: AttachmentResult{
				Title:    "Generated image",
				Filename: "image.png",
				MimeType: "image/png",
				Width:    job.Width,
				Height:   job.Height,
			},
			WorkflowID:     w.ID,
			WorkflowHash:   w.Hash,
			Profile:        job.Profile,
			Seed:           job.Seed,
			AspectRatio:    job.AspectRatio,
			NegativePrompt: job.NegativePrompt,
			Warnings:       job.Warnings,
		}

		// Phase 1 (always sync): submit to ComfyUI. Failure is reported to the LLM immediately.
		promptID, err := submitToComfyUI(comfyBase, promptPayload)
		if err != nil {
			return "", err
		}

		if useAsync {
			// Create a pending attachment row so the frontend can show a placeholder.
			att, err := tctx.Store.CreatePendingAttachment(tctx.ToolCallID, tctx.ConversationID, "Generated image", "image.png", "image/png")
			if err != nil {
				return "", fmt.Errorf("server error: could not create pending attachment: %w — tell the user image generation could not be started due to a server error", err)
			}
			log.Printf("Image generation pending attachment_id=%d prompt_id=%q", att.ID, promptID)

			// Phase 2 (async): poll, download, finalise.
			go func(attID, convID int64, bg bool) {
				img, err := pollComfyUI(comfyBase, promptID, w.Output, w.Timeout)
				if err != nil {
					log.Printf("Image generation failed attachment_id=%d: %v", attID, err)
					tctx.Store.FailAttachment(attID, err.Error()) //nolint:errcheck
					tctx.Hub.BroadcastAttachmentError(convID, attID, "Image generation failed")
					return
				}
				diskPath, err := downloadAndSaveImage(tctx.DataDir, comfyBase, img)
				if err != nil {
					log.Printf("Image download failed attachment_id=%d: %v", attID, err)
					tctx.Store.FailAttachment(attID, err.Error()) //nolint:errcheck
					tctx.Hub.BroadcastAttachmentError(convID, attID, "Image could not be saved")
					return
				}
				if err := tctx.Store.FinaliseAttachment(attID, diskPath); err != nil {
					log.Printf("Image finalise failed attachment_id=%d: %v", attID, err)
					tctx.Hub.BroadcastAttachmentError(convID, attID, "Image could not be saved")
					return
				}
				if bg {
					if err := tctx.Store.SetConversationBackground(convID, attID); err != nil {
						log.Printf("Setting conversation background conversation_id=%d attachment_id=%d: %v", convID, attID, err)
					} else {
						tctx.Hub.BroadcastConversationBackground(convID, attID)
					}
				}
				finalAtt, err := tctx.Store.GetAttachment(attID)
				if err != nil {
					log.Printf("Image get-after-finalise failed attachment_id=%d: %v", attID, err)
					return
				}
				log.Printf("Image generation complete attachment_id=%d", attID)
				tctx.Hub.BroadcastAttachmentReady(finalAtt, bg)
			}(att.ID, tctx.ConversationID, args.Background)

			result.AttachmentID = att.ID
			result.Status = "pending"
			out, _ := json.Marshal(result)
			return string(out), nil
		}

		// Sync path: poll and download before returning.
		img, err := pollComfyUI(comfyBase, promptID, w.Output, w.Timeout)
		if err != nil {
			return "", err
		}
		diskPath, err := downloadAndSaveImage(tctx.DataDir, comfyBase, img)
		if err != nil {
			return "", err
		}
		att, err := tctx.Store.CreateAttachment(tctx.ToolCallID, tctx.ConversationID, "Generated image", "image.png", "image/png", diskPath)
		if err != nil {
			return "", fmt.Errorf("server error: could not record image in database (%w) — tell the user the image was generated but could not be saved due to a server error", err)
		}
		if args.Background {
			if err := tctx.Store.SetConversationBackground(tctx.ConversationID, att.ID); err != nil {
				log.Printf("Setting conversation background conversation_id=%d attachment_id=%d: %v", tctx.ConversationID, att.ID, err)
			} else {
				tctx.Hub.BroadcastConversationBackground(tctx.ConversationID, att.ID)
			}
		}
		result.AttachmentID = att.ID
		result.Background = args.Background
		out, _ := json.Marshal(result)
		return string(out), nil
	}
}

// imageGenFunction builds the model-facing generate_image definition. With a
// registry, profile and workflow_id are enums of what is actually configured;
// without one (ComfyUI not configured) they are plain strings.
//
// Optional fields are nullable. Some models (and the Codex Responses backend)
// fill in every parameter rather than omitting optional ones; null gives them
// a way to say "not set" that doesn't collide with the profile/workflow_id
// exclusivity check or override the workflow's default negative prompt.
func imageGenFunction(reg *comfyui.Registry) toolFunction {
	profile := map[string]any{
		"type":        []any{"string", "null"},
		"description": "Image generation profile. Null or omit to use the default.",
	}
	workflowID := map[string]any{
		"type":        []any{"string", "null"},
		"description": "Exact workflow to use. Only set this when the user asks for a specific workflow or model; otherwise leave it null and use profile.",
	}
	if reg != nil {
		var names []any
		var lines []string
		for _, p := range reg.Profiles() {
			names = append(names, p.Name)
			line := "- " + p.Name
			if p.Description != "" {
				line += ": " + p.Description
			}
			lines = append(lines, line)
		}
		profile["enum"] = append(names, nil)
		profile["description"] = fmt.Sprintf("Image generation profile. Null or omit to use the default (%s). Must be null when workflow_id is set.\n%s", reg.DefaultProfile, strings.Join(lines, "\n"))

		var ids []any
		lines = lines[:0]
		for _, w := range reg.Exposed() {
			ids = append(ids, w.ID)
			lines = append(lines, "- "+w.ID+": "+w.Description)
		}
		if len(ids) > 0 {
			workflowID["enum"] = append(ids, nil)
			workflowID["description"] = workflowID["description"].(string) + " Must be null when profile is set.\n" + strings.Join(lines, "\n")
		}
	}

	aspect := make([]any, len(comfyui.AspectRatios))
	for i, a := range comfyui.AspectRatios {
		aspect[i] = a
	}
	props := map[string]any{
		"prompt": map[string]any{
			"type":        "string",
			"description": "Detailed visual description of the image. Include subject, setting, art style, lighting, and mood. If the user wants text in the image, put the exact text in quotes.",
		},
		"profile":     profile,
		"workflow_id": workflowID,
		"aspect_ratio": map[string]any{
			"type":        "string",
			"enum":        aspect,
			"description": "Image shape. Default: square.",
		},
		"negative_prompt": map[string]any{
			"type":        []any{"string", "null"},
			"description": "Replaces the workflow's default negative prompt (things to keep out of the image). Usually null or omitted, which keeps the default. The negative prompt actually used is returned in the result, so to adjust it, start from that value — e.g. remove \"text\" or \"watermark\" if the user wants lettering, or add something that keeps appearing unwanted. Some workflows ignore negative prompts; the result then includes a warning.",
		},
		"seed": map[string]any{
			"type":        []any{"integer", "null"},
			"description": "Seed for reproducible results. Null, omitted or 0 for a random seed; the seed used is returned in the result.",
		},
		"background": map[string]any{
			"type":        "boolean",
			"description": "If true, set the generated image as the chat background instead of displaying it inline.",
		},
		"async": map[string]any{
			"type":        "boolean",
			"description": "If false, block the response until the image is ready. Default: true (image generates in the background while you continue responding).",
		},
	}
	return toolFunction{
		Name:        "generate_image",
		Description: "Generates an image via ComfyUI. Use to illustrate scenes, characters, or objects, or whenever the user asks for a picture. Write the prompt yourself — be descriptive about subject, art style, lighting, and mood. Prefer profile over workflow_id. The generated image is automatically displayed in the chat — do not describe or embed it in your text response.",
		Parameters: toolParam{
			Type:       "object",
			Properties: props,
			Required:   []string{"prompt"},
		},
	}
}

// viewImageWait bounds how long view_image waits for an image that is still
// generating; it stays well inside the chat response timeout.
const viewImageWait = 3 * time.Minute

// viewImageResult is what view_image returns. It deliberately has no
// attachment_id field: messages.go treats that as a newly created attachment
// and would render the image a second time.
type viewImageResult struct {
	ImageID int64  `json:"image_id"`
	Status  string `json:"status"`
}

// viewedImageID extracts the image a successful view_image result refers to,
// or 0 for errors and anything else.
func viewedImageID(result string) int64 {
	var r viewImageResult
	if json.Unmarshal([]byte(result), &r) != nil || r.Status != "shown" {
		return 0
	}
	return r.ImageID
}

// executeViewImage validates the request; the tool loop attaches the image
// itself as a follow-up user message, since tool results are text-only.
func executeViewImage(argsJSON string, tctx ToolContext) (string, error) {
	var args struct {
		ImageID int64 `json:"image_id"`
	}
	if argsJSON != "" {
		if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
			return "", fmt.Errorf("invalid args: %w", err)
		}
	}
	if !tctx.ModelVision {
		return "", fmt.Errorf("the current model cannot view images")
	}

	id := args.ImageID
	if id == 0 {
		atts, err := tctx.Store.ListAttachmentsByConversation(tctx.ConversationID)
		if err != nil {
			return "", fmt.Errorf("server error: could not list images: %w", err)
		}
		for i := len(atts) - 1; i >= 0; i-- {
			if atts[i].Source == "tool" && strings.HasPrefix(atts[i].MimeType, "image/") && atts[i].Status != "error" {
				id = atts[i].ID
				break
			}
		}
		if id == 0 {
			return "", fmt.Errorf("no images have been generated in this conversation")
		}
	}

	log.Printf("Viewing image attachment_id=%d conversation_id=%d", id, tctx.ConversationID)
	deadline := time.Now().Add(viewImageWait)
	for {
		att, err := tctx.Store.GetAttachment(id)
		if err != nil || att.ConversationID != tctx.ConversationID || !strings.HasPrefix(att.MimeType, "image/") {
			return "", fmt.Errorf("there is no image with image_id %d in this conversation", id)
		}
		switch att.Status {
		case "ready":
			out, _ := json.Marshal(viewImageResult{ImageID: id, Status: "shown"})
			return string(out), nil
		case "error":
			return "", fmt.Errorf("image %d failed to generate: %s", id, att.Error)
		}
		if time.Now().After(deadline) {
			return "", fmt.Errorf("image %d is still generating after %d seconds — try again later", id, int(viewImageWait/time.Second))
		}
		time.Sleep(comfyPollInterval)
	}
}
