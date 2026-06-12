---
name: android-emulator-control
description: Use to drive the Android emulator for visual verification, UI debugging, and behavioral checks via adb and uiautomator. Trigger when verifying Android UI changes in a target, screenshotting an app state, simulating taps/text/swipes, or reading logcat in a tight verify-iterate loop.
---

# Android Emulator Control

Recipes for controlling an Android emulator from zsh. Use these the same way you'd use the `chrome-devtools` CLI on the web side: tight loops of "make change → run → screenshot → verify".

## Prerequisites

- Android SDK + platform tools installed (typically via Android Studio).
- `$ANDROID_HOME` (or `$ANDROID_SDK_ROOT`) set, with `$ANDROID_HOME/platform-tools` and `$ANDROID_HOME/emulator` on `$PATH`.
- At least one AVD created.

## List AVDs and devices

```sh
# Available AVDs
emulator -list-avds

# Currently running devices/emulators
adb devices
```

## Boot an emulator

```sh
# Start a named AVD in the background
emulator -avd Pixel_8_API_34 -no-snapshot-load &

# Wait until it's fully booted
adb wait-for-device shell 'while [[ -z $(getprop sys.boot_completed) ]]; do sleep 1; done'
```

If the target's tooling defines an emulator task, prefer that.

## Build and install the app

```sh
# Build debug APK (run from the target's source directory)
./gradlew :app:assembleDebug

# Install (replaces existing)
adb install -r app/build/outputs/apk/debug/app-debug.apk

# Launch the main activity
adb shell am start -n com.<app>.items/.MainActivity
```

## Screenshots

```sh
# Capture the current screen as PNG
adb exec-out screencap -p > /tmp/android-screenshot.png

# Then use the Read tool on /tmp/android-screenshot.png to view it
```

Take screenshots aggressively when verifying visual changes. Compare before/after by saving with descriptive names.

## Tap, type, swipe (via input + uiautomator)

```sh
# Tap at point (x, y) — coordinates in pixels, top-left origin
adb shell input tap 540 1200

# Type text (no spaces — use %s for spaces)
adb shell input text "Hello%sworld"

# Swipe from (x1, y1) to (x2, y2) over <ms> milliseconds
adb shell input swipe 540 1500 540 500 300

# Press a hardware key
adb shell input keyevent KEYCODE_BACK
adb shell input keyevent KEYCODE_HOME
adb shell input keyevent KEYCODE_ENTER

# Dump the UI hierarchy (saved to the device, then pulled)
adb shell uiautomator dump /sdcard/window_dump.xml
adb pull /sdcard/window_dump.xml /tmp/window_dump.xml
```

For Compose semantics, the `uiautomator dump` exposes accessibility nodes and `testTag` values — use those rather than coordinates for stability.

## Logs

```sh
# Stream logcat filtered for our app
adb logcat --pid=$(adb shell pidof -s com.<app>.items)

# Or filter by tag
adb logcat -s App:V

# One-shot dump of recent log
adb logcat -d -t 5m
```

For a tight verify loop, use streaming logcat in a backgrounded `Bash` call (`run_in_background: true`) and grep its output as you interact.

## Reset state

```sh
# Clear app data (preserves install)
adb shell pm clear com.<app>.items

# Or uninstall + reinstall
adb uninstall com.<app>.items
adb install -r app/build/outputs/apk/debug/app-debug.apk
```

## Verify-iterate loop pattern

```sh
# 1. Make a code change
# 2. Rebuild + reinstall
./gradlew :app:installDebug

# 3. Launch
adb shell am start -n com.<app>.items/.MainActivity

# 4. Drive the UI to the state you want to verify
adb shell input tap ...

# 5. Screenshot
adb exec-out screencap -p > /tmp/after.png

# 6. Read /tmp/after.png to verify

# 7. Repeat
```

If you find yourself running this sequence three times, codify it as a task in the target's own tooling.

## When NOT to use this skill

- Pure unit tests (no instrumented Android dependency) — run those via `specify verify <target>` instead.
- One-off lookups that aren't tied to verifying a code change.
- Anything that has a faster path through `kotlin.test` and Compose's test rule.
