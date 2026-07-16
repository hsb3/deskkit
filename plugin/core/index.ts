// Public surface of the harness-pure core domain library (D2). Adapters (plugin/mcp,
// plugin/opencode, plugin/claude-plugin) import from here — this is the single seam.

export {
  discoverProfile,
  discoverKnowledgeDir,
  loadProfile,
  extractFrontmatter,
  indexTree,
  scalarString,
  profileScalar,
  type Profile,
} from "./profile.js";

export {
  PLACEHOLDER_RE,
  MissingPlaceholderError,
  substitute,
  pick,
  resolveValue,
} from "./substitute.js";

export {
  discoverSchema,
  defaultSchemaPath,
  loadSchemaObject,
  compileValidator,
  getValidator,
  validateProfile,
  loadAndValidateProfile,
  type ValidationResult,
} from "./schema.js";

export {
  DEFAULT_KNOWLEDGE_BUDGET,
  knowledgeIndex,
  type KnowledgeEntry,
  type KnowledgeIndex,
} from "./knowledge.js";

export {
  TOOLS,
  profileGet,
  profileValidate,
  templateRender,
  knowledgeIndexTool,
  type ToolDef,
  type ToolContext,
  type JsonSchema,
} from "./tools.js";
