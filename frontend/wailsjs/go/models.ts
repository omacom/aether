export namespace color {
	
	export class Adjustments {
	    vibrance: number;
	    saturation: number;
	    contrast: number;
	    brightness: number;
	    shadows: number;
	    highlights: number;
	    hueShift: number;
	    temperature: number;
	    tint: number;
	    gamma: number;
	    blackPoint: number;
	    whitePoint: number;
	
	    static createFrom(source: any = {}) {
	        return new Adjustments(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.vibrance = source["vibrance"];
	        this.saturation = source["saturation"];
	        this.contrast = source["contrast"];
	        this.brightness = source["brightness"];
	        this.shadows = source["shadows"];
	        this.highlights = source["highlights"];
	        this.hueShift = source["hueShift"];
	        this.temperature = source["temperature"];
	        this.tint = source["tint"];
	        this.gamma = source["gamma"];
	        this.blackPoint = source["blackPoint"];
	        this.whitePoint = source["whitePoint"];
	    }
	}

}

export namespace favorites {
	
	export class Favorite {
	    path: string;
	    type?: string;
	    data?: Record<string, any>;
	
	    static createFrom(source: any = {}) {
	        return new Favorite(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.type = source["type"];
	        this.data = source["data"];
	    }
	}

}

export namespace githubsource {
	
	export class ImageInfo {
	    name: string;
	    url: string;
	    size: number;
	    type: string;
	    path: string;
	    htmlURL: string;
	
	    static createFrom(source: any = {}) {
	        return new ImageInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.url = source["url"];
	        this.size = source["size"];
	        this.type = source["type"];
	        this.path = source["path"];
	        this.htmlURL = source["htmlURL"];
	    }
	}
	export class ListContentsResult {
	    items: ImageInfo[];
	
	    static createFrom(source: any = {}) {
	        return new ListContentsResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.items = this.convertValues(source["items"], ImageInfo);
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
	export class ThumbnailResult {
	    dataURL: string;
	    width: number;
	    height: number;
	
	    static createFrom(source: any = {}) {
	        return new ThumbnailResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.dataURL = source["dataURL"];
	        this.width = source["width"];
	        this.height = source["height"];
	    }
	}

}

export namespace icontheme {
	
	export class PreviewSample {
	    kind: string;
	    pngData: string;
	
	    static createFrom(source: any = {}) {
	        return new PreviewSample(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = source["kind"];
	        this.pngData = source["pngData"];
	    }
	}
	export class Selection {
	    mode: string;
	    id?: string;
	
	    static createFrom(source: any = {}) {
	        return new Selection(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mode = source["mode"];
	        this.id = source["id"];
	    }
	}
	export class ThemePreview {
	    themeId: string;
	    samples: PreviewSample[];
	
	    static createFrom(source: any = {}) {
	        return new ThemePreview(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.themeId = source["themeId"];
	        this.samples = this.convertValues(source["samples"], PreviewSample);
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
	export class ThemeSummary {
	    id: string;
	    name: string;
	    inherits?: string[];
	    origin: string;
	    hasPreview: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ThemeSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.inherits = source["inherits"];
	        this.origin = source["origin"];
	        this.hasPreview = source["hasPreview"];
	    }
	}

}

export namespace ipc {
	
	export class Request {
	    cmd: string;
	    path?: string;
	    mode?: string;
	    name?: string;
	    index?: number;
	    value?: string;
	    palette?: string[];
	    vibrance?: number;
	    saturation?: number;
	    contrast?: number;
	    brightness?: number;
	    shadows?: number;
	    highlights?: number;
	    hue_shift?: number;
	    temperature?: number;
	    tint?: number;
	    gamma?: number;
	    black_point?: number;
	    white_point?: number;
	    light_mode?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Request(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.cmd = source["cmd"];
	        this.path = source["path"];
	        this.mode = source["mode"];
	        this.name = source["name"];
	        this.index = source["index"];
	        this.value = source["value"];
	        this.palette = source["palette"];
	        this.vibrance = source["vibrance"];
	        this.saturation = source["saturation"];
	        this.contrast = source["contrast"];
	        this.brightness = source["brightness"];
	        this.shadows = source["shadows"];
	        this.highlights = source["highlights"];
	        this.hue_shift = source["hue_shift"];
	        this.temperature = source["temperature"];
	        this.tint = source["tint"];
	        this.gamma = source["gamma"];
	        this.black_point = source["black_point"];
	        this.white_point = source["white_point"];
	        this.light_mode = source["light_mode"];
	    }
	}
	export class Response {
	    ok: boolean;
	    error?: string;
	    palette?: string[];
	    light_mode?: boolean;
	    mode?: string;
	    wallpaper?: string;
	    data?: number[];
	
	    static createFrom(source: any = {}) {
	        return new Response(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.error = source["error"];
	        this.palette = source["palette"];
	        this.light_mode = source["light_mode"];
	        this.mode = source["mode"];
	        this.wallpaper = source["wallpaper"];
	        this.data = source["data"];
	    }
	}

}

export namespace main {
	
	export class ApplyThemeRequest {
	    palette: string[];
	    wallpaperPath: string;
	    wallpaperBlur: boolean;
	    lightMode: boolean;
	    additionalImages: string[];
	    extendedColors: Record<string, string>;
	    yozakuraRoles: Record<string, string>;
	    glass?: number;
	    clock?: yozakura.Clock;
	    settings: theme.Settings;
	    appOverrides: Record<string, any>;
	    iconTheme: icontheme.Selection;
	    themeName: string;
	    themeID: string;
	
	    static createFrom(source: any = {}) {
	        return new ApplyThemeRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.palette = source["palette"];
	        this.wallpaperPath = source["wallpaperPath"];
	        this.wallpaperBlur = source["wallpaperBlur"];
	        this.lightMode = source["lightMode"];
	        this.additionalImages = source["additionalImages"];
	        this.extendedColors = source["extendedColors"];
	        this.yozakuraRoles = source["yozakuraRoles"];
	        this.glass = source["glass"];
	        this.clock = this.convertValues(source["clock"], yozakura.Clock);
	        this.settings = this.convertValues(source["settings"], theme.Settings);
	        this.appOverrides = source["appOverrides"];
	        this.iconTheme = this.convertValues(source["iconTheme"], icontheme.Selection);
	        this.themeName = source["themeName"];
	        this.themeID = source["themeID"];
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
	export class ExportFavoritesRequest {
	    paths: string[];
	
	    static createFrom(source: any = {}) {
	        return new ExportFavoritesRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.paths = source["paths"];
	    }
	}
	export class ExportThemeRequest {
	    name: string;
	    includedApps: string[];
	    palette: string[];
	    wallpaperPath: string;
	    wallpaperBlur: boolean;
	    lightMode: boolean;
	    additionalImages: string[];
	    extendedColors: Record<string, string>;
	    yozakuraRoles: Record<string, string>;
	    glass?: number;
	    clock?: yozakura.Clock;
	    appOverrides: Record<string, any>;
	    iconTheme: icontheme.Selection;
	
	    static createFrom(source: any = {}) {
	        return new ExportThemeRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.includedApps = source["includedApps"];
	        this.palette = source["palette"];
	        this.wallpaperPath = source["wallpaperPath"];
	        this.wallpaperBlur = source["wallpaperBlur"];
	        this.lightMode = source["lightMode"];
	        this.additionalImages = source["additionalImages"];
	        this.extendedColors = source["extendedColors"];
	        this.yozakuraRoles = source["yozakuraRoles"];
	        this.glass = source["glass"];
	        this.clock = this.convertValues(source["clock"], yozakura.Clock);
	        this.appOverrides = source["appOverrides"];
	        this.iconTheme = this.convertValues(source["iconTheme"], icontheme.Selection);
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
	export class ExternalImportPreview {
	    has_external_theme: boolean;
	    has_colors: boolean;
	    has_wallpaper: boolean;
	    source_url: string;
	    palette?: string[];
	    wallpaper?: string;
	    theme_name?: string;
	    mode?: string;
	    edit: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ExternalImportPreview(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.has_external_theme = source["has_external_theme"];
	        this.has_colors = source["has_colors"];
	        this.has_wallpaper = source["has_wallpaper"];
	        this.source_url = source["source_url"];
	        this.palette = source["palette"];
	        this.wallpaper = source["wallpaper"];
	        this.theme_name = source["theme_name"];
	        this.mode = source["mode"];
	        this.edit = source["edit"];
	    }
	}
	export class ExtractFromImagesResult {
	    palette: string[];
	    skipped: number;
	
	    static createFrom(source: any = {}) {
	        return new ExtractFromImagesResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.palette = source["palette"];
	        this.skipped = source["skipped"];
	    }
	}
	export class ImportResult {
	    colors: string[];
	    extendedColors: Record<string, string>;
	    yozakuraRoles: Record<string, string>;
	    glass?: number;
	    clock?: yozakura.Clock;
	    name: string;
	    path: string;
	    wallpaperPath: string;
	    wallpaperBlur: boolean;
	    lightMode: boolean;
	    iconTheme: icontheme.Selection;
	
	    static createFrom(source: any = {}) {
	        return new ImportResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.colors = source["colors"];
	        this.extendedColors = source["extendedColors"];
	        this.yozakuraRoles = source["yozakuraRoles"];
	        this.glass = source["glass"];
	        this.clock = this.convertValues(source["clock"], yozakura.Clock);
	        this.name = source["name"];
	        this.path = source["path"];
	        this.wallpaperPath = source["wallpaperPath"];
	        this.wallpaperBlur = source["wallpaperBlur"];
	        this.lightMode = source["lightMode"];
	        this.iconTheme = this.convertValues(source["iconTheme"], icontheme.Selection);
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
	export class ImportYozakuraResult {
	    imported: string[];
	    default: string;
	    error: string;
	
	    static createFrom(source: any = {}) {
	        return new ImportYozakuraResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.imported = source["imported"];
	        this.default = source["default"];
	        this.error = source["error"];
	    }
	}
	export class SaveAndApplyThemeRequest {
	    name: string;
	    updateExisting: boolean;
	    palette: string[];
	    wallpaperPath: string;
	    wallpaperBlur: boolean;
	    lightMode: boolean;
	    additionalImages: string[];
	    extendedColors: Record<string, string>;
	    yozakuraRoles: Record<string, string>;
	    glass?: number;
	    clock?: yozakura.Clock;
	    settings: theme.Settings;
	    appOverrides: Record<string, any>;
	    iconTheme: icontheme.Selection;
	    themeName: string;
	    themeID: string;
	
	    static createFrom(source: any = {}) {
	        return new SaveAndApplyThemeRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.updateExisting = source["updateExisting"];
	        this.palette = source["palette"];
	        this.wallpaperPath = source["wallpaperPath"];
	        this.wallpaperBlur = source["wallpaperBlur"];
	        this.lightMode = source["lightMode"];
	        this.additionalImages = source["additionalImages"];
	        this.extendedColors = source["extendedColors"];
	        this.yozakuraRoles = source["yozakuraRoles"];
	        this.glass = source["glass"];
	        this.clock = this.convertValues(source["clock"], yozakura.Clock);
	        this.settings = this.convertValues(source["settings"], theme.Settings);
	        this.appOverrides = source["appOverrides"];
	        this.iconTheme = this.convertValues(source["iconTheme"], icontheme.Selection);
	        this.themeName = source["themeName"];
	        this.themeID = source["themeID"];
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
	export class SaveBlueprintRequest {
	    name: string;
	    palette: string[];
	    wallpaperPath: string;
	    wallpaperBlur: boolean;
	    lightMode: boolean;
	    additionalImages: string[];
	    lockedColors: number[];
	    extendedColors: Record<string, string>;
	    yozakuraRoles: Record<string, string>;
	    glass?: number;
	    clock?: yozakura.Clock;
	    appOverrides: Record<string, any>;
	    adjustments: Record<string, number>;
	    iconTheme: icontheme.Selection;
	    matchWallpapers: string[];
	
	    static createFrom(source: any = {}) {
	        return new SaveBlueprintRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.palette = source["palette"];
	        this.wallpaperPath = source["wallpaperPath"];
	        this.wallpaperBlur = source["wallpaperBlur"];
	        this.lightMode = source["lightMode"];
	        this.additionalImages = source["additionalImages"];
	        this.lockedColors = source["lockedColors"];
	        this.extendedColors = source["extendedColors"];
	        this.yozakuraRoles = source["yozakuraRoles"];
	        this.glass = source["glass"];
	        this.clock = this.convertValues(source["clock"], yozakura.Clock);
	        this.appOverrides = source["appOverrides"];
	        this.adjustments = source["adjustments"];
	        this.iconTheme = this.convertValues(source["iconTheme"], icontheme.Selection);
	        this.matchWallpapers = source["matchWallpapers"];
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
	export class SyncStateRequest {
	    palette: string[];
	    wallpaperPath: string;
	    wallpaperBlur: boolean;
	    lightMode: boolean;
	    extendedColors: Record<string, string>;
	    yozakuraRoles: Record<string, string>;
	    glass?: number;
	    clock?: yozakura.Clock;
	    appOverrides: Record<string, any>;
	    additionalImages: string[];
	    iconTheme: icontheme.Selection;
	
	    static createFrom(source: any = {}) {
	        return new SyncStateRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.palette = source["palette"];
	        this.wallpaperPath = source["wallpaperPath"];
	        this.wallpaperBlur = source["wallpaperBlur"];
	        this.lightMode = source["lightMode"];
	        this.extendedColors = source["extendedColors"];
	        this.yozakuraRoles = source["yozakuraRoles"];
	        this.glass = source["glass"];
	        this.clock = this.convertValues(source["clock"], yozakura.Clock);
	        this.appOverrides = source["appOverrides"];
	        this.additionalImages = source["additionalImages"];
	        this.iconTheme = this.convertValues(source["iconTheme"], icontheme.Selection);
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
	export class TargetInfo {
	    id: string;
	    label: string;
	    detected: boolean;
	    path: string;
	
	    static createFrom(source: any = {}) {
	        return new TargetInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.label = source["label"];
	        this.detected = source["detected"];
	        this.path = source["path"];
	    }
	}
	export class YozakuraRolesRequest {
	    palette: string[];
	    extendedColors: Record<string, string>;
	    lightMode: boolean;
	    overrides: Record<string, string>;
	    glass?: number;
	    clock?: yozakura.Clock;
	
	    static createFrom(source: any = {}) {
	        return new YozakuraRolesRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.palette = source["palette"];
	        this.extendedColors = source["extendedColors"];
	        this.lightMode = source["lightMode"];
	        this.overrides = source["overrides"];
	        this.glass = source["glass"];
	        this.clock = this.convertValues(source["clock"], yozakura.Clock);
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
	export class YozakuraRolesResult {
	    roles: Record<string, string>;
	    explicit: Record<string, boolean>;
	    glass: number;
	    clock: yozakura.Clock;
	    contrast: yozakura.PairResult[];
	    accentCandidates: string[];
	
	    static createFrom(source: any = {}) {
	        return new YozakuraRolesResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.roles = source["roles"];
	        this.explicit = source["explicit"];
	        this.glass = source["glass"];
	        this.clock = this.convertValues(source["clock"], yozakura.Clock);
	        this.contrast = this.convertValues(source["contrast"], yozakura.PairResult);
	        this.accentCandidates = source["accentCandidates"];
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

}

export namespace targets {
	
	export class Status {
	    id: string;
	    label: string;
	    state: string;
	    path: string;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new Status(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.label = source["label"];
	        this.state = source["state"];
	        this.path = source["path"];
	        this.error = source["error"];
	    }
	}

}

export namespace template {
	
	export class ColorRoles {
	    background: string;
	    foreground: string;
	    black: string;
	    red: string;
	    green: string;
	    yellow: string;
	    blue: string;
	    magenta: string;
	    cyan: string;
	    white: string;
	    bright_black: string;
	    bright_red: string;
	    bright_green: string;
	    bright_yellow: string;
	    bright_blue: string;
	    bright_magenta: string;
	    bright_cyan: string;
	    bright_white: string;
	    accent: string;
	    cursor: string;
	    selection_foreground: string;
	    selection_background: string;
	
	    static createFrom(source: any = {}) {
	        return new ColorRoles(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.background = source["background"];
	        this.foreground = source["foreground"];
	        this.black = source["black"];
	        this.red = source["red"];
	        this.green = source["green"];
	        this.yellow = source["yellow"];
	        this.blue = source["blue"];
	        this.magenta = source["magenta"];
	        this.cyan = source["cyan"];
	        this.white = source["white"];
	        this.bright_black = source["bright_black"];
	        this.bright_red = source["bright_red"];
	        this.bright_green = source["bright_green"];
	        this.bright_yellow = source["bright_yellow"];
	        this.bright_blue = source["bright_blue"];
	        this.bright_magenta = source["bright_magenta"];
	        this.bright_cyan = source["bright_cyan"];
	        this.bright_white = source["bright_white"];
	        this.accent = source["accent"];
	        this.cursor = source["cursor"];
	        this.selection_foreground = source["selection_foreground"];
	        this.selection_background = source["selection_background"];
	    }
	}

}

export namespace theme {
	
	export class ApplyResult {
	    success: boolean;
	    themePath: string;
	    targets: targets.Status[];
	
	    static createFrom(source: any = {}) {
	        return new ApplyResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.themePath = source["themePath"];
	        this.targets = this.convertValues(source["targets"], targets.Status);
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
	    includeZed: boolean;
	    includeVscode: boolean;
	    includeNeovim: boolean;
	    selectedNeovimConfig: string;
	    includedApps?: Record<string, boolean>;
	    excludedApps?: Record<string, boolean>;
	
	    static createFrom(source: any = {}) {
	        return new Settings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.includeZed = source["includeZed"];
	        this.includeVscode = source["includeVscode"];
	        this.includeNeovim = source["includeNeovim"];
	        this.selectedNeovimConfig = source["selectedNeovimConfig"];
	        this.includedApps = source["includedApps"];
	        this.excludedApps = source["excludedApps"];
	    }
	}
	export class StateSnapshot {
	    palette: string[];
	    wallpaperPath: string;
	    wallpaperBlur: boolean;
	    lightMode: boolean;
	    lockedColors: Record<number, boolean>;
	    colorRoles: template.ColorRoles;
	    extendedColors: Record<string, string>;
	    yozakuraRoles: Record<string, string>;
	    glass?: number;
	    clock?: yozakura.Clock;
	    extractionMode: string;
	    additionalImages: string[];
	    appOverrides: Record<string, any>;
	    iconTheme: icontheme.Selection;
	
	    static createFrom(source: any = {}) {
	        return new StateSnapshot(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.palette = source["palette"];
	        this.wallpaperPath = source["wallpaperPath"];
	        this.wallpaperBlur = source["wallpaperBlur"];
	        this.lightMode = source["lightMode"];
	        this.lockedColors = source["lockedColors"];
	        this.colorRoles = this.convertValues(source["colorRoles"], template.ColorRoles);
	        this.extendedColors = source["extendedColors"];
	        this.yozakuraRoles = source["yozakuraRoles"];
	        this.glass = source["glass"];
	        this.clock = this.convertValues(source["clock"], yozakura.Clock);
	        this.extractionMode = source["extractionMode"];
	        this.additionalImages = source["additionalImages"];
	        this.appOverrides = source["appOverrides"];
	        this.iconTheme = this.convertValues(source["iconTheme"], icontheme.Selection);
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

}

export namespace wallhaven {
	
	export class SearchMeta {
	    current_page: number;
	    last_page: number;
	    total: number;
	    seed?: string;
	
	    static createFrom(source: any = {}) {
	        return new SearchMeta(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.current_page = source["current_page"];
	        this.last_page = source["last_page"];
	        this.total = source["total"];
	        this.seed = source["seed"];
	    }
	}
	export class SearchParams {
	    q: string;
	    categories: string;
	    purity: string;
	    sorting: string;
	    order: string;
	    page: number;
	    atleast: string;
	    colors: string;
	
	    static createFrom(source: any = {}) {
	        return new SearchParams(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.q = source["q"];
	        this.categories = source["categories"];
	        this.purity = source["purity"];
	        this.sorting = source["sorting"];
	        this.order = source["order"];
	        this.page = source["page"];
	        this.atleast = source["atleast"];
	        this.colors = source["colors"];
	    }
	}
	export class WallpaperInfo {
	    id: string;
	    url: string;
	    path: string;
	    resolution: string;
	    file_size: number;
	    category: string;
	    purity: string;
	    thumbs: Record<string, string>;
	
	    static createFrom(source: any = {}) {
	        return new WallpaperInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.url = source["url"];
	        this.path = source["path"];
	        this.resolution = source["resolution"];
	        this.file_size = source["file_size"];
	        this.category = source["category"];
	        this.purity = source["purity"];
	        this.thumbs = source["thumbs"];
	    }
	}
	export class SearchResult {
	    data: WallpaperInfo[];
	    meta: SearchMeta;
	
	    static createFrom(source: any = {}) {
	        return new SearchResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.data = this.convertValues(source["data"], WallpaperInfo);
	        this.meta = this.convertValues(source["meta"], SearchMeta);
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

}

export namespace wallpaper {
	
	export class WallpaperInfo {
	    path: string;
	    name: string;
	    size: number;
	    modTime: number;
	
	    static createFrom(source: any = {}) {
	        return new WallpaperInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.name = source["name"];
	        this.size = source["size"];
	        this.modTime = source["modTime"];
	    }
	}

}

export namespace yozakura {
	
	export class Clock {
	    digits: string;
	    colon: string;
	    glow: number;
	    ornament: string;
	
	    static createFrom(source: any = {}) {
	        return new Clock(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.digits = source["digits"];
	        this.colon = source["colon"];
	        this.glow = source["glow"];
	        this.ornament = source["ornament"];
	    }
	}
	export class PairResult {
	    pair: string;
	    ratio: number;
	    min: number;
	    state: string;
	
	    static createFrom(source: any = {}) {
	        return new PairResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.pair = source["pair"];
	        this.ratio = source["ratio"];
	        this.min = source["min"];
	        this.state = source["state"];
	    }
	}

}

