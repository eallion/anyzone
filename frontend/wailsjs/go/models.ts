export namespace config {
	
	export class Account {
	    id: string;
	    name: string;
	    provider: string;
	    encrypted_credentials: string;
	    custom_proxy?: string;
	    // Go type: time
	    created_at: any;
	
	    static createFrom(source: any = {}) {
	        return new Account(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.provider = source["provider"];
	        this.encrypted_credentials = source["encrypted_credentials"];
	        this.custom_proxy = source["custom_proxy"];
	        this.created_at = this.convertValues(source["created_at"], null);
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
	export class Settings {
	    is_pin_configured: boolean;
	    pin_length?: number;
	    pin_salt?: string;
	    pin_verifier?: string;
	    auto_lock_minutes: number;
	    proxy_enabled: boolean;
	    proxy_url: string;
	    proxy_routing: string;
	    theme: string;
	    aggregate_all_default: boolean;
	    zone_page_size: number;
	    auto_hide_mismatched_ns: boolean;
	    hidden_zones: string[];
	
	    static createFrom(source: any = {}) {
	        return new Settings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.is_pin_configured = source["is_pin_configured"];
	        this.pin_length = source["pin_length"];
	        this.pin_salt = source["pin_salt"];
	        this.pin_verifier = source["pin_verifier"];
	        this.auto_lock_minutes = source["auto_lock_minutes"];
	        this.proxy_enabled = source["proxy_enabled"];
	        this.proxy_url = source["proxy_url"];
	        this.proxy_routing = source["proxy_routing"];
	        this.theme = source["theme"];
	        this.aggregate_all_default = source["aggregate_all_default"];
	        this.zone_page_size = source["zone_page_size"];
	        this.auto_hide_mismatched_ns = source["auto_hide_mismatched_ns"];
	        this.hidden_zones = source["hidden_zones"];
	    }
	}

}

export namespace dnsutil {
	
	export class NSCheckResult {
	    account_id?: string;
	    expected_provider?: string;
	    domain: string;
	    actual_ns: string[];
	    is_matched: boolean;
	    detected_provider: string;
	    status: string;
	    error_message?: string;
	
	    static createFrom(source: any = {}) {
	        return new NSCheckResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.account_id = source["account_id"];
	        this.expected_provider = source["expected_provider"];
	        this.domain = source["domain"];
	        this.actual_ns = source["actual_ns"];
	        this.is_matched = source["is_matched"];
	        this.detected_provider = source["detected_provider"];
	        this.status = source["status"];
	        this.error_message = source["error_message"];
	    }
	}

}

export namespace main {
	
	export class ZoneCheckReq {
	    account_id?: string;
	    domain: string;
	    provider: string;
	
	    static createFrom(source: any = {}) {
	        return new ZoneCheckReq(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.account_id = source["account_id"];
	        this.domain = source["domain"];
	        this.provider = source["provider"];
	    }
	}

}

export namespace provider {
	
	export class Record {
	    id: string;
	    zone_id: string;
	    zone_name: string;
	    type: string;
	    name: string;
	    content: string;
	    ttl: number;
	    priority?: number;
	    proxied?: boolean;
	    comment?: string;
	
	    static createFrom(source: any = {}) {
	        return new Record(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.zone_id = source["zone_id"];
	        this.zone_name = source["zone_name"];
	        this.type = source["type"];
	        this.name = source["name"];
	        this.content = source["content"];
	        this.ttl = source["ttl"];
	        this.priority = source["priority"];
	        this.proxied = source["proxied"];
	        this.comment = source["comment"];
	    }
	}
	export class Zone {
	    id: string;
	    name: string;
	    status: string;
	    provider: string;
	    account_id: string;
	    account_name?: string;
	    actual_ns?: string[];
	    ns_status?: string;
	    detected_provider?: string;
	    is_hidden?: boolean;
	    access_type?: string;
	
	    static createFrom(source: any = {}) {
	        return new Zone(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.status = source["status"];
	        this.provider = source["provider"];
	        this.account_id = source["account_id"];
	        this.account_name = source["account_name"];
	        this.actual_ns = source["actual_ns"];
	        this.ns_status = source["ns_status"];
	        this.detected_provider = source["detected_provider"];
	        this.is_hidden = source["is_hidden"];
	        this.access_type = source["access_type"];
	    }
	}

}

