#include <stdlib.h>
#include <string.h>
#include <wayland-client.h>
#include "wayland-protocols/ext-idle-notify-v1-client-protocol.h"

// Include protocol implementation once
#include "wayland-protocols/ext-idle-notify-v1-protocol.c"

// Global state
static struct wl_display *display = NULL;
static struct wl_registry *registry = NULL;
static struct ext_idle_notifier_v1 *idle_notifier = NULL;
static struct wl_seat *seat = NULL;
static struct ext_idle_notification_v1 *notification = NULL;

// External Go callbacks
extern void goIdleCallback();
extern void goResumeCallback();

// C callback handlers
static void handle_idle(void *data, struct ext_idle_notification_v1 *notification) {
goIdleCallback();
}

static void handle_resume(void *data, struct ext_idle_notification_v1 *notification) {
goResumeCallback();
}

static const struct ext_idle_notification_v1_listener idle_notification_listener = {
.idled = handle_idle,
.resumed = handle_resume,
};

// Registry listener
static void registry_handle_global(void *data, struct wl_registry *registry,
uint32_t name, const char *interface, uint32_t version) {
if (strcmp(interface, ext_idle_notifier_v1_interface.name) == 0) {
idle_notifier = wl_registry_bind(registry, name, 
&ext_idle_notifier_v1_interface, 1);
} else if (strcmp(interface, "wl_seat") == 0 && seat == NULL) {
seat = wl_registry_bind(registry, name, &wl_seat_interface, 1);
}
}

static void registry_handle_global_remove(void *data, struct wl_registry *registry, uint32_t name) {
}

static const struct wl_registry_listener registry_listener = {
.global = registry_handle_global,
.global_remove = registry_handle_global_remove,
};

// API functions
int wayland_cgo_init() {
display = wl_display_connect(NULL);
if (!display) {
return -1;
}

registry = wl_display_get_registry(display);
if (!registry) {
wl_display_disconnect(display);
return -2;
}

wl_registry_add_listener(registry, &registry_listener, NULL);

wl_display_roundtrip(display);
wl_display_roundtrip(display);

if (!idle_notifier) {
wl_display_disconnect(display);
return -3;
}

if (!seat) {
wl_display_disconnect(display);
return -4;
}

return 0;
}

int wayland_cgo_register_timeout(uint32_t timeout_ms) {
if (!idle_notifier || !seat) {
return -1;
}

notification = ext_idle_notifier_v1_get_idle_notification(
idle_notifier, timeout_ms, seat);

if (!notification) {
return -2;
}

ext_idle_notification_v1_add_listener(notification,
&idle_notification_listener, NULL);

wl_display_roundtrip(display);
return 0;
}

int wayland_cgo_dispatch() {
if (!display) {
return -1;
}

if (wl_display_dispatch(display) == -1) {
return -2;
}

return 0;
}

void wayland_cgo_cleanup() {
if (notification) {
ext_idle_notification_v1_destroy(notification);
notification = NULL;
}
if (idle_notifier) {
ext_idle_notifier_v1_destroy(idle_notifier);
idle_notifier = NULL;
}
if (registry) {
wl_registry_destroy(registry);
registry = NULL;
}
if (display) {
wl_display_disconnect(display);
display = NULL;
}
}
