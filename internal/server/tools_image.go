package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type comfyImage struct {
	Filename  string `json:"filename"`
	Subfolder string `json:"subfolder"`
	Type      string `json:"type"`
}

func submitToComfyUI(comfyBase string, promptPayload []byte) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
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

func pollComfyUI(comfyBase, promptID string) (*comfyImage, error) {
	type comfyOutput struct {
		Images []comfyImage `json:"images"`
	}
	type comfyJob struct {
		Outputs map[string]comfyOutput `json:"outputs"`
	}
	deadline := time.Now().Add(120 * time.Second)
	for time.Now().Before(deadline) {
		time.Sleep(1 * time.Second)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
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
		if job, ok := history[promptID]; ok {
			for _, output := range job.Outputs {
				if len(output.Images) > 0 {
					img := output.Images[0]
					return &img, nil
				}
			}
		}
	}
	return nil, fmt.Errorf("image generation timed out after 120 seconds — tell the user the image was not produced in time and suggest checking that ComfyUI is processing jobs correctly")
}

func downloadAndSaveImage(dataDir, comfyBase string, img *comfyImage) (string, error) {
	viewURL := comfyBase + "/view?filename=" + url.QueryEscape(img.Filename) +
		"&subfolder=" + url.QueryEscape(img.Subfolder) +
		"&type=" + url.QueryEscape(img.Type)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
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

// makeImageExecutor returns an executor for a ComfyUI workflow. comfyURL,
// workflowFile, defaultSteps, and defaultCFG are captured at startup.
func makeImageExecutor(comfyURL, workflowFile string, defaultSteps int, defaultCFG float64) func(string, ToolContext) (string, error) {
	return func(argsJSON string, tctx ToolContext) (string, error) {
		var args struct {
			Prompt         string  `json:"prompt"`
			NegativePrompt string  `json:"negative_prompt"`
			Seed           int64   `json:"seed"`
			CFG            float64 `json:"cfg"`
			Width          int     `json:"width"`
			Height         int     `json:"height"`
			Steps          int     `json:"steps"`
			Background     bool    `json:"background"`
			Async          *bool   `json:"async"` // nil = default true
		}
		if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
			return "", fmt.Errorf("invalid args: %w", err)
		}
		if args.Prompt == "" {
			return "", fmt.Errorf("prompt is required")
		}

		seed := args.Seed
		if seed <= 0 {
			seed = rand.Int63()
		}
		cfg := args.CFG
		if cfg <= 0 {
			cfg = defaultCFG
		}
		width := args.Width
		if width <= 0 {
			width = 1024
		}
		height := args.Height
		if height <= 0 {
			height = 1024
		}
		steps := args.Steps
		if steps <= 0 {
			steps = defaultSteps
		}

		workflowData, err := os.ReadFile(workflowFile)
		if err != nil {
			return "", fmt.Errorf("generate_image: cannot read workflow file %q — check the workflow path in lemon.toml: %w", workflowFile, err)
		}

		// Replace placeholders: prompts use JSON-encoded strings; numeric values are bare.
		// Placeholders absent from the workflow JSON are replaced as no-ops.
		promptJSON, _ := json.Marshal(args.Prompt)
		negJSON, _ := json.Marshal(args.NegativePrompt)
		workflowStr := strings.ReplaceAll(string(workflowData), `"__PROMPT__"`, string(promptJSON))
		workflowStr = strings.ReplaceAll(workflowStr, `"__NEGATIVE_PROMPT__"`, string(negJSON))
		workflowStr = strings.ReplaceAll(workflowStr, `__SEED__`, strconv.FormatInt(seed, 10))
		workflowStr = strings.ReplaceAll(workflowStr, `__CFG__`, strconv.FormatFloat(cfg, 'f', -1, 64))
		workflowStr = strings.ReplaceAll(workflowStr, `__WIDTH__`, strconv.Itoa(width))
		workflowStr = strings.ReplaceAll(workflowStr, `__HEIGHT__`, strconv.Itoa(height))
		workflowStr = strings.ReplaceAll(workflowStr, `__STEPS__`, strconv.Itoa(steps))

		var workflow map[string]any
		if err := json.Unmarshal([]byte(workflowStr), &workflow); err != nil {
			return "", fmt.Errorf("workflow file %q contains invalid JSON: %w — tell the user the image could not be generated due to an invalid workflow file", workflowFile, err)
		}

		clientID := randomID()
		promptPayload, _ := json.Marshal(map[string]any{
			"prompt":    workflow,
			"client_id": clientID,
		})

		comfyBase := strings.TrimRight(comfyURL, "/")
		useAsync := args.Async == nil || *args.Async

		log.Printf("Generating image prompt=%q async=%v conversation_id=%d", args.Prompt, useAsync, tctx.ConversationID)

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
				img, err := pollComfyUI(comfyBase, promptID)
				if err != nil {
					log.Printf("Image generation failed attachment_id=%d: %v", attID, err)
					tctx.Store.FailAttachment(attID, err.Error()) //nolint:errcheck
					tctx.Hub.BroadcastAttachmentError(convID, attID, "Image generation timed out")
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

			result := AttachmentResult{
				AttachmentID: att.ID,
				Title:        "Generated image",
				Filename:     "image.png",
				MimeType:     "image/png",
				Status:       "pending",
			}
			out, _ := json.Marshal(result)
			return string(out), nil
		}

		// Sync path: poll and download before returning.
		img, err := pollComfyUI(comfyBase, promptID)
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
		result := AttachmentResult{
			AttachmentID: att.ID,
			Title:        "Generated image",
			Filename:     "image.png",
			MimeType:     "image/png",
			Background:   args.Background,
		}
		out, _ := json.Marshal(result)
		return string(out), nil
	}
}
