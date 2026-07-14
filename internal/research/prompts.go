package research

// Prompts ported verbatim from the deep research spec (docs/deep_research_spec.html).
// The algorithm is modelled on Alibaba's IterResearch / Tongyi DeepResearch approach.

const slugPrompt = `Create a short filename slug for this research job.

**Research title or question:** %s

Return ONLY a descriptive slug with these constraints:
- 2-4 meaningful words
- lowercase ASCII letters, digits, and underscores only
- 32 characters maximum
- no generic words such as research, report, investigate, or analysis

Example: house_building_limited_res`

const researchPlanPrompt = `You are a research strategist. Before searching, analyze this question and create a research plan.

**Question:** %s

Break this question down:
1. What are the key sub-topics that need to be covered for a comprehensive answer?
2. What specific data points, facts, or perspectives should we look for?
3. What would a complete, high-quality answer include?

Return a JSON object with:
- "sub_questions": Array of 3-6 specific sub-questions to investigate
- "key_topics": Array of key topics/angles to cover
- "success_criteria": One sentence describing what a complete answer looks like

Example:
{
  "sub_questions": ["What is the cost of living in X?", "How is the healthcare system?"],
  "key_topics": ["economy", "healthcare", "safety", "culture"],
  "success_criteria": "A balanced comparison covering cost, quality of life, and practical considerations."
}`

const classifyPrompt = `Classify this research question into exactly ONE category.
Categories: product, comparison, howto, factcheck
If none fit well, respond with: general

Question: %s

Respond with ONLY the category name, nothing else.`

const brainstormClassifyPrompt = `Classify this brainstorm brief into exactly ONE output format.

Formats:
- design-doc: Designing or building one specific solution, product, or system — a clear single recommendation
- options: Exploring multiple distinct alternatives or approaches to compare — no single winner expected
- ideas: Open-ended creative brainstorm — a variety of concepts to spark thinking, not a single solution
- analysis: Strategic, philosophical, or conceptual investigation — exploring a question deeply for insight
- explainer: Factual explanation — what something is, how it works, or why it matters

Brief: %s

Respond with ONLY the format name (e.g. design-doc), nothing else.`

const queryGenPrompt = `You are a research assistant planning web searches.

**Original question:** %s

**Research plan:**
%s

**What we know so far:**
%s

**Round:** %d

Generate %d focused search queries that will help answer the question.
%s

Return ONLY a JSON array of query strings, nothing else.
Example: ["query one", "query two", "query three"]`

const queryGenFirstRoundInstruction = "This is the first round — generate broad, diverse queries that explore the key facets of the question."

const queryGenFollowUpInstruction = "We already have partial findings. Generate targeted follow-up queries to fill gaps, verify claims, or explore specific aspects that the report doesn't yet cover well."

// Creative instructions are used for the bonus "extra effort" rounds that run
// after the report would normally be considered complete.
const queryGenCreativeInstruction = "We already have substantial findings and the report looks reasonably complete. This is a bonus round — think laterally. Generate creative, non-obvious queries that approach the question from fresh angles: adjacent topics, contrarian viewpoints, recent developments, edge cases, or expert sources the report has not yet explored. Avoid repeating earlier searches."

const queryGenVeryCreativeInstruction = "This is a final, deep bonus round — be bold and highly creative. Generate unconventional, exploratory queries that dig into surprising angles, niche or primary sources, dissenting opinions, historical context, or future implications. Deliberately seek out perspectives and information the report is still missing. Avoid anything resembling earlier searches."

// ── Brainstorm mode ──────────────────────────────────────────
//
// Brainstorm mode flips the research pipeline around: instead of web search
// driving the process, the model invents and develops ideas, reaching for the
// web only when it decides it needs to look something up. These prompts replace
// their research-mode counterparts when Config.Mode == ModeBrainstorm.

const brainstormPlanPrompt = `You are a creative strategist about to brainstorm and design solutions to a problem. Before generating ideas, frame the problem.

**Brief:** %s

Think about:
1. What is actually being asked for — what would a great outcome look like?
2. What are the key dimensions, constraints, or angles worth exploring?
3. What different directions could a strong solution take?

Write a short framing (a few sentences, no headings) that captures the goal, the main constraints, and 2-4 promising directions to explore. This will guide the ideation that follows. Do not solve the problem yet — just frame it.`

const brainstormQueryPrompt = `You are developing ideas for a creative/design brief. Decide whether you need to look anything up on the web this round.

**Brief:** %s

**Framing:**
%s

**Ideas so far:**
%s

**Round:** %d

%s

You can rely on your own knowledge for most of the work. Only search when external information would genuinely improve the result — e.g. to check a fact, find prior art or real-world examples, get concrete numbers, or verify something you are unsure about.

If you need to search, return a JSON array of 1-3 focused search query strings.
If you can make good progress from your own knowledge this round, return an empty array: []

Return ONLY the JSON array, nothing else.
Example (search): ["existing solutions to X", "typical cost of Y"]
Example (no search): []`

// brainstormForcedQueryPrompt is used for the first round when the user has
// asked for at least one web search — it removes the "return []" escape hatch
// and requires the model to commit to one or more queries.
const brainstormForcedQueryPrompt = `You are developing ideas for a creative/design brief, and you should ground this round in some real-world research.

**Brief:** %s

**Framing:**
%s

**Ideas so far:**
%s

**Round:** %d

%s

Find external information that will sharpen the work — e.g. prior art, real-world examples, concrete numbers, or facts worth verifying.

Return a JSON array of 1-3 focused search query strings. Do not return an empty array.

Return ONLY the JSON array, nothing else.
Example: ["existing solutions to X", "typical cost of Y"]`

const brainstormForcedSearchInstruction = "Ground this round in real-world research: identify what is worth looking up to strengthen the ideas."

const brainstormSearchInstruction = "Keep developing the ideas. Search only if a specific fact, example, or number would meaningfully strengthen them."

const brainstormCreativeInstruction = "This is a bonus round — push into fresh, less obvious directions. Search only if grounding a new angle in real-world examples or data would help."

const brainstormVeryCreativeInstruction = "This is a final, bold bonus round — explore surprising, unconventional directions. Search only if a concrete external reference would sharpen a new idea."

const brainstormDevelopPrompt = `You are an inventor and designer developing ideas for a brief.

**Brief:** %s

**Framing:**
%s

**Design notes so far:**
%s

**New research from this round:**
%s

Develop the work further this round: introduce new ideas where useful, deepen the most promising ones, address weaknesses and trade-offs, and fold in any new research. Keep the strongest material, prune dead ends, and maintain a clear, logical structure. Where a web source informed a specific point, cite the source ID inline, e.g. [S1].

Write only the updated design notes — no preamble or meta-commentary.`

const brainstormStopPrompt = `You are deciding whether a brainstorming/design effort is well developed enough.

**Brief:** %s

**Design notes so far:**
%s

**Rounds completed:** %d of %d

Consider:
- Is there at least one strong, well-developed concept (or a solid set of options)?
- Are the key design decisions and trade-offs worked out?
- Would another round of ideation meaningfully improve the result, or is it diminishing returns?

If rounds completed is well below the target, prefer continuing unless the ideas are already strong and thoroughly developed.

Reply with ONLY "YES" or "NO" followed by a brief one-sentence reason.
Example: "YES — There is a strong, fully worked-out concept with trade-offs addressed."
Example: "NO — The ideas are still thin and the trade-offs are unexplored."`

const brainstormFinalPrompt = `Write a polished **design document** for this brief, based on the ideas developed below.

**Brief:** %s

**Framing:**
%s

**Developed design notes:**
%s

**Supporting web findings:**
%s

Structure the document with clear ## headings, roughly:
- A short **executive summary** of the proposed solution
- **The concept** — the core idea(s), explained clearly
- **How it works** — the key components, mechanics, or design decisions
- **Trade-offs and alternatives** — what was chosen and why, and what was rejected
- **Risks and open questions**
- **Next steps** — concrete actions to move forward

Be specific and decisive — recommend, don't just list options. Where a web source informed a specific fact, cite the source ID inline, e.g. [S1]. Write in an engaging, clear style. Length should fit the idea: thorough but not padded.`

// extractorSystem is the goal-based content extraction prompt
// (credited to the Tongyi DeepResearch lineage).
const extractorSystem = `Extract relevant information from a webpage for a given research goal.

Goal: %s

Task guidelines:
1. Locate the specific sections directly related to the goal within the provided webpage content.
2. Identify and extract the most relevant information; output full original context where possible, up to three or more paragraphs.
3. Organize into a concise paragraph with logical flow, judging each piece of information's contribution to the goal.

Respond in JSON with exactly these fields: "rational", "evidence", "summary".

Example:
{
    "rational": "This section discusses X which directly relates to the goal of understanding Y",
    "evidence": "Full quotes and context from the page...",
    "summary": "Concise summary of how this information answers the goal"
}`

const evidenceLedgerPrompt = `You maintain compact structured research memory. Update the evidence ledger using the new findings.

**Original question:** %s

**Current evidence ledger (JSON, or legacy notes to convert):**
%s

**New findings from this round:**
%s

Return the COMPLETE updated ledger as one JSON object with exactly this shape:
{
  "claims": [
    {
      "claim": "one precise factual or analytical claim",
      "supporting_sources": ["S1"],
      "contradicting_sources": ["S2"],
      "confidence": "high|medium|low|uncertain"
    }
  ],
  "assumptions": ["an assumption that materially affects the answer"],
  "calculations": ["formula, inputs, result, units, and source IDs where applicable"],
  "unresolved_gaps": [
    {"question": "one concrete unanswered question", "notes": "why it matters and what evidence is missing"}
  ]
}

Rules:
- This is research memory, not report prose. Be terse and factual.
- Merge duplicate claims; update their source lists and confidence instead of repeating them.
- Preserve disagreements by listing contradicting source IDs rather than forcing consensus.
- Use only source IDs present in the supplied findings or current ledger. Write IDs as S1, not [S1].
- Retain still-relevant prior claims, assumptions, calculations, and gaps.
- Remove a gap only when the evidence now answers it.
- Do not include markdown, a preamble, or keys outside the schema.`

const stopPrompt = `You are deciding whether a research report is comprehensive enough.

**Original question:** %s

**Current report:**
%s

**Rounds completed:** %d of %d

Based on the report so far, do we have enough information to answer the question comprehensively?  Consider:
- Are the key aspects of the question addressed?
- Are there obvious gaps or unanswered sub-questions?
- Is the evidence sufficient and from multiple sources?

If rounds completed is well below the target, prefer continuing unless the report is already exhaustive.

Reply with ONLY "YES" or "NO" followed by a brief one-sentence reason.
Example: "YES — The report covers all major aspects with evidence from multiple sources."
Example: "NO — We still lack information about the economic impact."`

const finalReportPrompt = `Write a **long, detailed, comprehensive** research report answering this question:

**Question:** %s

**Structured evidence ledger:**
%s

**Raw collected findings with source IDs:**
%s

Requirements:
- Write at MINIMUM 1500 words — this should be a thorough, magazine-quality article
- Use clear ## headings and ### subheadings to organize into logical sections
- Each section should have multiple detailed paragraphs, not just bullet points
- Synthesize and analyze the information — explain WHY things matter, draw comparisons, provide context
- Include specific data points, numbers, and statistics from the evidence
- Cite source IDs inline for specific claims, e.g. [S1]. Do not invent source IDs; use only IDs present in the raw findings.
- Note where sources agree and where they disagree
- Add a brief executive summary at the top
- End with a clear conclusion that directly answers the question
- Write in an engaging, informative style — not dry or robotic`

const expandReportPrompt = `This report is too brief. Please expand it significantly:
- Add detailed paragraphs for each section (not just bullet points)
- Include specific data, numbers, and comparisons from the evidence
- Explain context and significance — don't just list facts
- Use ## headings and ### subheadings
- Preserve and add source ID citations such as [S1] for sourced claims
- Target at least 1000 words
Write the full expanded report now.`

// ── Section-based deep report ─────────────────────────────────
//
// Used when DeepReport is set. The report is built section by section from the
// raw findings instead of in one summarising pass. Research- and brainstorm-mode
// variants mirror the single-shot final prompts above.

const outlineDraftPrompt = `You are planning the structure of a detailed research report.

**Question:** %s

**Research plan:**
%s

**Structured evidence ledger:**
%s

**Collected evidence (raw findings):**
%s

Design an outline as a set of sections that together answer the question thoroughly and without overlap. Cover every important aspect supported by the evidence. Aim for roughly 4-8 substantial sections. Do NOT include an executive summary or conclusion section — those are added separately.%s

Return ONLY a JSON object of this exact shape, nothing else:
{"sections": [{"title": "Section title", "intent": "One sentence on what this section covers and why"}]}`

const outlineRefinePrompt = `You are reviewing and improving the outline for a detailed research report before it is written.

**Question:** %s

**Research plan:**
%s

**Collected evidence (raw findings):**
%s

**Draft outline:**
%s

Critically review the draft outline against the question, the plan, and the evidence:
- Add sections for anything important that is missing or under-covered.
- Remove or merge sections that are redundant or thin.
- Order the sections so the report flows logically.
Do not include an executive summary or conclusion section (those are added separately).

Return ONLY the improved outline as a JSON object of the same shape, nothing else:
{"sections": [{"title": "Section title", "intent": "One sentence on what this section covers and why"}]}`

const sectionWritePrompt = `You are writing ONE section of a detailed, long-form research report. Other sections are written separately.

**Question:** %s

**Research plan:**
%s

**This section:** %s
**What this section should cover:** %s

**Full report outline (for context — avoid overlapping with the other sections):**
%s

**Structured evidence ledger:**
%s

**Collected evidence (raw findings — draw on the relevant ones):**
%s

Write this section in depth: multiple detailed paragraphs with specific facts, numbers, and comparisons drawn from the evidence, explaining why things matter. Cite source IDs inline for specific claims, e.g. [S1]. Do not invent source IDs; use only IDs present in the raw findings. Start with a "## " heading for this section's title. Write only this section's content — no preamble, no other sections.`

const reportGluePartPrompt = `You are writing one part of a detailed research report.

**Question:** %s

**Report section outline:**
%s

**Structured evidence ledger:**
%s

Write %s

Write only that text — no heading, no preamble, no meta-commentary.`

const brainstormOutlineDraftPrompt = `You are planning the structure of a detailed design document.

**Brief:** %s

**Framing:**
%s

**Design notes so far:**
%s

**Any supporting research:**
%s

Design an outline as a set of sections that fully present and justify the proposed solution. Typical sections include the core concept, how it works, key components or mechanics, and trade-offs and alternatives — but tailor them to this brief. Aim for roughly 4-8 substantial sections. Do NOT include an executive summary or next-steps section — those are added separately.

Return ONLY a JSON object of this exact shape, nothing else:
{"sections": [{"title": "Section title", "intent": "One sentence on what this section covers and why"}]}`

const brainstormOutlineRefinePrompt = `You are reviewing and improving the outline for a design document before it is written.

**Brief:** %s

**Framing:**
%s

**Any supporting research:**
%s

**Draft outline:**
%s

Critically review the draft outline against the brief and the design work so far:
- Add sections for anything important that is missing or under-developed.
- Remove or merge sections that are redundant or thin.
- Order the sections so the document flows logically.
Do not include an executive summary or next-steps section (those are added separately).

Return ONLY the improved outline as a JSON object of the same shape, nothing else:
{"sections": [{"title": "Section title", "intent": "One sentence on what this section covers and why"}]}`

const brainstormSectionWritePrompt = `You are writing ONE section of a detailed design document. Other sections are written separately.

**Brief:** %s

**Framing:**
%s

**This section:** %s
**What this section should cover:** %s

**Full document outline (for context — avoid overlapping with the other sections):**
%s

**Design notes so far:**
%s

**Any supporting research (draw on the relevant items):**
%s

Develop this section in depth. Be specific and decisive — recommend, don't just list options. Where a web source informed a specific point, cite the source ID inline, e.g. [S1]. Do not invent source IDs; use only IDs present in the supporting research. Start with a "## " heading for this section's title. Write only this section's content — no preamble, no other sections.`

const brainstormGluePartPrompt = `You are writing one part of a design document.

**Brief:** %s

**Document section outline:**
%s

**Overview of the design:**
%s

Write %s

Write only that text — no heading, no preamble, no meta-commentary.`

// categoryPrompts are appended to the final-report prompt when a category
// was detected or supplied.
var categoryPrompts = map[string]string{
	"product": `

IMPORTANT FORMAT OVERRIDE — this is a PRODUCT research report:
- Structure as a RANKED LIST of products/options (best first)
- For EACH product include: name as ### heading, approximate price, 2-3 sentence summary, **Pros:** bullet list, **Cons:** bullet list, **Where to buy:** URLs as links
- Start with a quick-compare markdown table of top picks (columns: Name, Price, Best For, Rating)
- End with a ## Verdict section picking Best Overall and Best Value
- Still include source ID citations inline`,
	"comparison": `

IMPORTANT FORMAT OVERRIDE — this is a COMPARISON report:
- Create a ## Comparison Table as a markdown table comparing ALL options across key criteria (rows = criteria, columns = options)
- Use checkmarks, ratings, or short values in cells
- Write a ## section per option with its strengths, weaknesses, and ideal use case
- End with ## Best For verdicts (e.g., "**Best for small teams:** Option A because...")
- Include a ## Shared Considerations section for things that apply to all options`,
	"howto": `

IMPORTANT FORMAT OVERRIDE — this is a HOW-TO guide:
- Start with ## Quick Guide — a super concise numbered list (one line per step, no details, just the action). Example: 1. Install X  2. Run Y  3. Configure Z
- Then ## Prerequisites listing what's needed before starting
- Then the detailed steps: ## Step 1: ..., ## Step 2: ...
- Each step should have a clear heading and detailed instructions
- Use blockquotes (> ) for tips and warnings: > **Tip:** ... or > **Warning:** ...
- End with ## Common Mistakes section
- Add estimated time and difficulty level near the top`,
	"factcheck": `

IMPORTANT FORMAT OVERRIDE — this is a FACT-CHECK report:
- Start with ## The Claim restating what's being checked
- Create ## Evidence For and ## Evidence Against sections
- Each piece of evidence should be a ### with source name, what it found, and how strong the evidence is
- Include a ## Verdict section with one of: **Supported**, **Mixed Evidence**, or **Unsupported**
- End with ## Nuance & Caveats for important context and limitations
- Be balanced and cite source IDs for every claim`,
}

// brainstormFormatOverrides are appended to brainstormFinalPrompt (and to the
// outline draft prompt for deep-report runs) when a specific output format was
// detected. The default format (design-doc) uses the base prompt unchanged.
var brainstormFormatOverrides = map[string]string{
	"options": `

IMPORTANT FORMAT OVERRIDE — present as a set of OPTIONS, not a single recommended design:
- Open with a brief ## Brief framing the question and what a good option looks like
- Present 3–5 distinct options, each as a ## [Option Name] section
- For each option: describe the core concept, how it works, its strengths and weaknesses, and when to choose it
- End with a ## Which to choose section giving honest guidance on which options suit which situations
- Do NOT pick a single winner — help the reader make the right choice for their context`,

	"ideas": `

IMPORTANT FORMAT OVERRIDE — present as a curated IDEAS document, not a single design:
- Open with a brief ## Overview framing the creative space and what kinds of ideas are possible
- Present 5–8 developed ideas, each as a ### [Idea Name] section
- For each idea: a one-sentence hook, how it works, what makes it interesting, and any obvious challenges
- Ideas should be meaningfully distinct — avoid overlap
- End with a ## Directions section suggesting which ideas might combine well or are most worth pursuing first`,

	"analysis": `

IMPORTANT FORMAT OVERRIDE — write a strategic/philosophical ANALYSIS, not a design document:
- Use a discursive essay structure with ## headings for the main threads of inquiry
- Explore the question from multiple angles: context, competing perspectives, underlying tensions, implications
- Aim for insight and clarity rather than a prescriptive plan — the goal is understanding, not a deliverable
- Do NOT include a Next steps section or a single prescriptive recommendation unless the brief asks for one
- Cite any web sources inline using source IDs such as [S1]`,

	"explainer": `

IMPORTANT FORMAT OVERRIDE — write a clear EXPLAINER, not a design document:
- Open with a direct, concise answer to the question (1–3 sentences)
- Use ## sections to build out the explanation with depth and context
- Use concrete examples, analogies, and plain language — avoid jargon
- Only include as much depth as the question warrants — do not pad
- Do NOT include Next steps, trade-offs, or risks sections unless directly relevant to the explanation`,
}

// brainstormFormatConclusion maps output format to the deep-report glue
// conclusion heading and instruction.
var brainstormFormatConclusion = map[string][2]string{
	"design-doc": {"## Next steps", "a 'Next steps' list of concrete, actionable recommendations to move the design forward."},
	"options":    {"## Which to choose", "a 'Which to choose' section giving honest guidance on which options suit which situations and contexts."},
	"ideas":      {"## Directions", "a 'Directions' section suggesting which ideas might combine well or are most worth pursuing first."},
	"analysis":   {"## Synthesis", "a 'Synthesis' section that ties the main threads of the analysis together into a clear takeaway."},
	"explainer":  {"## Summary", "a concise 'Summary' section that recaps the key points of the explanation in plain language."},
}

// lowQualityMarkers — a finding whose summary contains any of these
// (case-insensitive) is discarded.
var lowQualityMarkers = []string{
	"insufficient to",
	"content is insufficient",
	"no substantive data",
	"does not contain",
	"not relevant to",
	"no relevant information",
	"unable to extract",
	"completely unrelated",
	"boilerplate",
	"footer text",
	"cookie consent",
	"cookie banner",
	"cookie notice",
	"copyright notice",
	"copyright footer",
	"all rights reserved",
}
