// @ts-check
// Wails runtime bindings

export function EventsOnMultiple(eventName, callback, maxCallbacks) {
    window.runtime.EventsOnMultiple(eventName, callback, maxCallbacks);
}

export function EventsOn(eventName, callback) {
    EventsOnMultiple(eventName, callback, -1);
}

export function EventsOff(eventName) {
    return window.runtime.EventsOff(eventName);
}

export function EventsOffAll() {
    return window.runtime.EventsOffAll();
}

export function EventsEmit(eventName) {
    let args = [eventName].slice.call(arguments);
    return window.runtime.EventsEmit.apply(null, args);
}

export function BrowserOpenURL(url) {
    window.runtime.BrowserOpenURL(url);
}

export function Environment() {
    return window.runtime.Environment();
}

export function Quit() {
    window.runtime.Quit();
}
