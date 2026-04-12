// Wails runtime declarations

export function EventsOnMultiple(eventName: string, callback: (...args: any[]) => void, maxCallbacks: number): void;
export function EventsOn(eventName: string, callback: (...args: any[]) => void): void;
export function EventsOff(eventName: string): void;
export function EventsOffAll(): void;
export function EventsEmit(eventName: string, ...args: any[]): void;
export function BrowserOpenURL(url: string): void;
export function Environment(): Promise<{buildType: string; platform: string; arch: string}>;
export function Quit(): void;
