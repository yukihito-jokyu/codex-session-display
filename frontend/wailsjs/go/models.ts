export namespace dto {
	
	export class TokenBreakdown {
	    total_tokens: number;
	    input_tokens: number;
	    output_tokens: number;
	    reasoning_output_tokens: number;
	
	    static createFrom(source: any = {}) {
	        return new TokenBreakdown(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.total_tokens = source["total_tokens"];
	        this.input_tokens = source["input_tokens"];
	        this.output_tokens = source["output_tokens"];
	        this.reasoning_output_tokens = source["reasoning_output_tokens"];
	    }
	}
	export class TimelineItemDetail {
	    label: string;
	    value: string;
	
	    static createFrom(source: any = {}) {
	        return new TimelineItemDetail(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.label = source["label"];
	        this.value = source["value"];
	    }
	}
	export class ConversationTimelineItem {
	    selection_id: string;
	    node_id: string;
	    node_ids: string[];
	    token_count_indices: number[];
	    kind: string;
	    label: string;
	    role: string;
	    body: string;
	    timestamp: string;
	    record_count: number;
	    collapsible: boolean;
	    details: TimelineItemDetail[];
	    last_token_usage: TokenBreakdown;
	    token_count_count: number;
	    total_token_usage?: TokenBreakdown;
	
	    static createFrom(source: any = {}) {
	        return new ConversationTimelineItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.selection_id = source["selection_id"];
	        this.node_id = source["node_id"];
	        this.node_ids = source["node_ids"];
	        this.token_count_indices = source["token_count_indices"];
	        this.kind = source["kind"];
	        this.label = source["label"];
	        this.role = source["role"];
	        this.body = source["body"];
	        this.timestamp = source["timestamp"];
	        this.record_count = source["record_count"];
	        this.collapsible = source["collapsible"];
	        this.details = this.convertValues(source["details"], TimelineItemDetail);
	        this.last_token_usage = this.convertValues(source["last_token_usage"], TokenBreakdown);
	        this.token_count_count = source["token_count_count"];
	        this.total_token_usage = this.convertValues(source["total_token_usage"], TokenBreakdown);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ConversationTimelineTurn {
	    index: number;
	    turn_id: string;
	    pseudo: boolean;
	    duration_ms: number;
	    consumed_tokens: TokenBreakdown;
	    items: ConversationTimelineItem[];
	
	    static createFrom(source: any = {}) {
	        return new ConversationTimelineTurn(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.index = source["index"];
	        this.turn_id = source["turn_id"];
	        this.pseudo = source["pseudo"];
	        this.duration_ms = source["duration_ms"];
	        this.consumed_tokens = this.convertValues(source["consumed_tokens"], TokenBreakdown);
	        this.items = this.convertValues(source["items"], ConversationTimelineItem);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class FlowEdge {
	    id: string;
	    source: string;
	    target: string;
	    type: string;
	    animated: boolean;
	
	    static createFrom(source: any = {}) {
	        return new FlowEdge(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.source = source["source"];
	        this.target = source["target"];
	        this.type = source["type"];
	        this.animated = source["animated"];
	    }
	}
	export class TokenBadgeData {
	    consumedTokens: number;
	    tokenCountIndex: number;
	    boundCount: number;
	
	    static createFrom(source: any = {}) {
	        return new TokenBadgeData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.consumedTokens = source["consumedTokens"];
	        this.tokenCountIndex = source["tokenCountIndex"];
	        this.boundCount = source["boundCount"];
	    }
	}
	export class NodeData {
	    category: string;
	    label: string;
	    icon: string;
	    summary: string;
	    fullText?: string;
	    meta?: Record<string, any>;
	    batchIndex?: number;
	    batchSize?: number;
	    collapsed?: boolean;
	    textLength?: number;
	    turnIndex?: number;
	    tokenBadge?: TokenBadgeData;
	
	    static createFrom(source: any = {}) {
	        return new NodeData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.category = source["category"];
	        this.label = source["label"];
	        this.icon = source["icon"];
	        this.summary = source["summary"];
	        this.fullText = source["fullText"];
	        this.meta = source["meta"];
	        this.batchIndex = source["batchIndex"];
	        this.batchSize = source["batchSize"];
	        this.collapsed = source["collapsed"];
	        this.textLength = source["textLength"];
	        this.turnIndex = source["turnIndex"];
	        this.tokenBadge = this.convertValues(source["tokenBadge"], TokenBadgeData);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Position {
	    x: number;
	    y: number;
	
	    static createFrom(source: any = {}) {
	        return new Position(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.x = source["x"];
	        this.y = source["y"];
	    }
	}
	export class FlowNode {
	    id: string;
	    type: string;
	    position: Position;
	    data: NodeData;
	
	    static createFrom(source: any = {}) {
	        return new FlowNode(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.type = source["type"];
	        this.position = this.convertValues(source["position"], Position);
	        this.data = this.convertValues(source["data"], NodeData);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	
	export class SubagentDetail {
	    id: string;
	    nickname: string;
	    total_tokens: number;
	    input_tokens: number;
	    output_tokens: number;
	
	    static createFrom(source: any = {}) {
	        return new SubagentDetail(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.nickname = source["nickname"];
	        this.total_tokens = source["total_tokens"];
	        this.input_tokens = source["input_tokens"];
	        this.output_tokens = source["output_tokens"];
	    }
	}
	export class TokenCountEntry {
	    index: number;
	    turn_index: number;
	    bound_to_node_id: string;
	    model_context_window: number;
	    last_token_usage?: model.TokenDetail;
	    total_token_usage?: model.TokenDetail;
	
	    static createFrom(source: any = {}) {
	        return new TokenCountEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.index = source["index"];
	        this.turn_index = source["turn_index"];
	        this.bound_to_node_id = source["bound_to_node_id"];
	        this.model_context_window = source["model_context_window"];
	        this.last_token_usage = this.convertValues(source["last_token_usage"], model.TokenDetail);
	        this.total_token_usage = this.convertValues(source["total_token_usage"], model.TokenDetail);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class TurnStatistics {
	    index: number;
	    collaboration_mode_kind: string;
	    duration_ms: number;
	    time_to_first_token_ms: number;
	    token_count_count: number;
	    consumed_tokens: TokenBreakdown;
	
	    static createFrom(source: any = {}) {
	        return new TurnStatistics(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.index = source["index"];
	        this.collaboration_mode_kind = source["collaboration_mode_kind"];
	        this.duration_ms = source["duration_ms"];
	        this.time_to_first_token_ms = source["time_to_first_token_ms"];
	        this.token_count_count = source["token_count_count"];
	        this.consumed_tokens = this.convertValues(source["consumed_tokens"], TokenBreakdown);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Statistics {
	    duration_ms: number;
	    total_tokens: number;
	    tool_call_count: number;
	    token_count_count: number;
	    context_window_size: number;
	    turn_count: number;
	    turns: TurnStatistics[];
	
	    static createFrom(source: any = {}) {
	        return new Statistics(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.duration_ms = source["duration_ms"];
	        this.total_tokens = source["total_tokens"];
	        this.tool_call_count = source["tool_call_count"];
	        this.token_count_count = source["token_count_count"];
	        this.context_window_size = source["context_window_size"];
	        this.turn_count = source["turn_count"];
	        this.turns = this.convertValues(source["turns"], TurnStatistics);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class SessionDetailResponse {
	    id: string;
	    cache_schema_version: number;
	    parsed_at: string;
	    nodes: FlowNode[];
	    edges: FlowEdge[];
	    statistics: Statistics;
	    token_counts: TokenCountEntry[];
	    timeline: ConversationTimelineTurn[];
	    parent_session_id?: string;
	    child_session_ids?: string[];
	    subagents?: SubagentDetail[];
	
	    static createFrom(source: any = {}) {
	        return new SessionDetailResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.cache_schema_version = source["cache_schema_version"];
	        this.parsed_at = source["parsed_at"];
	        this.nodes = this.convertValues(source["nodes"], FlowNode);
	        this.edges = this.convertValues(source["edges"], FlowEdge);
	        this.statistics = this.convertValues(source["statistics"], Statistics);
	        this.token_counts = this.convertValues(source["token_counts"], TokenCountEntry);
	        this.timeline = this.convertValues(source["timeline"], ConversationTimelineTurn);
	        this.parent_session_id = source["parent_session_id"];
	        this.child_session_ids = source["child_session_ids"];
	        this.subagents = this.convertValues(source["subagents"], SubagentDetail);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class SessionSummary {
	    id: string;
	    file_path: string;
	    cwd?: string;
	    cli_version?: string;
	    originator?: string;
	    model_provider?: string;
	    branch?: string;
	    source?: string;
	    timestamp?: string;
	    file_size: number;
	    file_modified_at?: string;
	    parsed: boolean;
	    parent_session_id?: string;
	    child_session_ids?: string[];
	    total_tokens?: number;
	    input_tokens?: number;
	    output_tokens?: number;
	    reasoning_tokens?: number;
	    turn_count?: number;
	    step_count?: number;
	
	    static createFrom(source: any = {}) {
	        return new SessionSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.file_path = source["file_path"];
	        this.cwd = source["cwd"];
	        this.cli_version = source["cli_version"];
	        this.originator = source["originator"];
	        this.model_provider = source["model_provider"];
	        this.branch = source["branch"];
	        this.source = source["source"];
	        this.timestamp = source["timestamp"];
	        this.file_size = source["file_size"];
	        this.file_modified_at = source["file_modified_at"];
	        this.parsed = source["parsed"];
	        this.parent_session_id = source["parent_session_id"];
	        this.child_session_ids = source["child_session_ids"];
	        this.total_tokens = source["total_tokens"];
	        this.input_tokens = source["input_tokens"];
	        this.output_tokens = source["output_tokens"];
	        this.reasoning_tokens = source["reasoning_tokens"];
	        this.turn_count = source["turn_count"];
	        this.step_count = source["step_count"];
	    }
	}
	
	
	
	
	
	
	
	export class UpdateResult {
	    hasUpdate: boolean;
	    current: string;
	    latest: string;
	    releaseUrl: string;
	    downloadUrl: string;
	
	    static createFrom(source: any = {}) {
	        return new UpdateResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.hasUpdate = source["hasUpdate"];
	        this.current = source["current"];
	        this.latest = source["latest"];
	        this.releaseUrl = source["releaseUrl"];
	        this.downloadUrl = source["downloadUrl"];
	    }
	}

}

export namespace model {
	
	export class TokenDetail {
	    total_tokens: number;
	    input_tokens: number;
	    output_tokens: number;
	    reasoning_output_tokens: number;
	    cached_input_tokens: number;
	
	    static createFrom(source: any = {}) {
	        return new TokenDetail(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.total_tokens = source["total_tokens"];
	        this.input_tokens = source["input_tokens"];
	        this.output_tokens = source["output_tokens"];
	        this.reasoning_output_tokens = source["reasoning_output_tokens"];
	        this.cached_input_tokens = source["cached_input_tokens"];
	    }
	}

}

