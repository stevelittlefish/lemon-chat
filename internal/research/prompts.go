package research

// Prompts ported verbatim from the deep research spec (docs/deep_research_spec.html).
// The algorithm is modelled on Alibaba's IterResearch / Tongyi DeepResearch approach.

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

const synthesizePrompt = `You are updating an evolving research report.

**Original question:** %s

**Current report:**
%s

**New findings from this round:**
%s

Integrate the new findings into the existing report. Produce an updated, well-organized report that answers the original question as completely as possible given all evidence so far. Remove redundancy, resolve contradictions, and maintain logical flow. Keep source URLs as inline citations where relevant.

Write only the updated report — no preamble or meta-commentary.`

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

**All collected evidence and analysis:**
%s

Requirements:
- Write at MINIMUM 1500 words — this should be a thorough, magazine-quality article
- Use clear ## headings and ### subheadings to organize into logical sections
- Each section should have multiple detailed paragraphs, not just bullet points
- Synthesize and analyze the information — explain WHY things matter, draw comparisons, provide context
- Include specific data points, numbers, and statistics from the evidence
- Include source URLs as inline citations [like this](url)
- Note where sources agree and where they disagree
- Add a brief executive summary at the top
- End with a clear conclusion that directly answers the question
- Write in an engaging, informative style — not dry or robotic`

const expandReportPrompt = `This report is too brief. Please expand it significantly:
- Add detailed paragraphs for each section (not just bullet points)
- Include specific data, numbers, and comparisons from the evidence
- Explain context and significance — don't just list facts
- Use ## headings and ### subheadings
- Target at least 1000 words
Write the full expanded report now.`

// categoryPrompts are appended to the final-report prompt when a category
// was detected or supplied.
var categoryPrompts = map[string]string{
	"product": `

IMPORTANT FORMAT OVERRIDE — this is a PRODUCT research report:
- Structure as a RANKED LIST of products/options (best first)
- For EACH product include: name as ### heading, approximate price, 2-3 sentence summary, **Pros:** bullet list, **Cons:** bullet list, **Where to buy:** URLs as links
- Start with a quick-compare markdown table of top picks (columns: Name, Price, Best For, Rating)
- End with a ## Verdict section picking Best Overall and Best Value
- Still include source citations inline`,
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
- Be balanced and cite sources for every claim`,
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
