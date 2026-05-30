export namespace dto {
	
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
	    }
	}

}

