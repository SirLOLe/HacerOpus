export namespace main {
	
	export class ConversionSettings {
	    bitrate: string;
	    coverSize: number;
	    quality: number;
	    replaceCover: boolean;
	    compressionLevel: number;
	    threads: number;
	
	    static createFrom(source: any = {}) {
	        return new ConversionSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.bitrate = source["bitrate"];
	        this.coverSize = source["coverSize"];
	        this.quality = source["quality"];
	        this.replaceCover = source["replaceCover"];
	        this.compressionLevel = source["compressionLevel"];
	        this.threads = source["threads"];
	    }
	}

}

