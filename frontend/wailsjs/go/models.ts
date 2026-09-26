export namespace main {
	
	export class Config {
	    host: string;
	    user: string;
	    password: string;
	    labels: string[];
	    saveDir: string;
	    syncMinutes: number;
	    density: string;
	    scale: number;
	
	    static createFrom(source: any = {}) {
	        return new Config(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.host = source["host"];
	        this.user = source["user"];
	        this.password = source["password"];
	        this.labels = source["labels"];
	        this.saveDir = source["saveDir"];
	        this.syncMinutes = source["syncMinutes"];
	        this.density = source["density"];
	        this.scale = source["scale"];
	    }
	}
	export class Count {
	    key: string;
	    name: string;
	    total: number;
	    unread: number;
	
	    static createFrom(source: any = {}) {
	        return new Count(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.name = source["name"];
	        this.total = source["total"];
	        this.unread = source["unread"];
	    }
	}
	export class Export {
	    path: string;
	    markdown: string;
	
	    static createFrom(source: any = {}) {
	        return new Export(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.markdown = source["markdown"];
	    }
	}
	export class Highlight {
	    id: number;
	    messageId: number;
	    text: string;
	    prefix: string;
	    suffix: string;
	    note: string;
	    created: number;
	    subject?: string;
	    fromName?: string;
	    date?: number;
	
	    static createFrom(source: any = {}) {
	        return new Highlight(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.messageId = source["messageId"];
	        this.text = source["text"];
	        this.prefix = source["prefix"];
	        this.suffix = source["suffix"];
	        this.note = source["note"];
	        this.created = source["created"];
	        this.subject = source["subject"];
	        this.fromName = source["fromName"];
	        this.date = source["date"];
	    }
	}
	export class Issue {
	    id: number;
	    fromName: string;
	    fromAddr: string;
	    subject: string;
	    date: number;
	    html?: string;
	    words: number;
	    read: boolean;
	    highlights: Highlight[];
	
	    static createFrom(source: any = {}) {
	        return new Issue(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.fromName = source["fromName"];
	        this.fromAddr = source["fromAddr"];
	        this.subject = source["subject"];
	        this.date = source["date"];
	        this.html = source["html"];
	        this.words = source["words"];
	        this.read = source["read"];
	        this.highlights = this.convertValues(source["highlights"], Highlight);
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
	export class Msg {
	    id: number;
	    fromName: string;
	    fromAddr: string;
	    subject: string;
	    date: number;
	    html?: string;
	    words: number;
	    read: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Msg(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.fromName = source["fromName"];
	        this.fromAddr = source["fromAddr"];
	        this.subject = source["subject"];
	        this.date = source["date"];
	        this.html = source["html"];
	        this.words = source["words"];
	        this.read = source["read"];
	    }
	}
	export class Query {
	    label: string;
	    sender: string;
	    search: string;
	    sort: string;
	    offset: number;
	
	    static createFrom(source: any = {}) {
	        return new Query(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.label = source["label"];
	        this.sender = source["sender"];
	        this.search = source["search"];
	        this.sort = source["sort"];
	        this.offset = source["offset"];
	    }
	}

}

