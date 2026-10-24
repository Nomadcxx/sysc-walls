#!/usr/bin/env python3
"""Run a fullscreen effects timeline in one Kitty window, without changing config."""
import argparse
import json
import math
import shlex
import signal
import subprocess
import time
from pathlib import Path


def parse_timeline(text):
    scenes = []
    for number, line in enumerate(text.splitlines(), 1):
        fields = shlex.split(line, comments=True)
        if not fields:
            continue
        try:
            seconds = float(fields[0])
        except ValueError:
            raise ValueError(f"line {number}: duration must be a number") from None
        if not math.isfinite(seconds) or seconds <= 0 or len(fields) < 3:
            raise ValueError(f"line {number}: expected positive seconds, effect and theme")
        scenes.append((seconds, fields[1:]))
    if not scenes:
        raise ValueError("timeline is empty")
    return scenes


def stop(child):
    if child.poll() is None:
        child.terminate()
        try:
            child.wait(timeout=3)
        except subprocess.TimeoutExpired:
            child.kill()
            child.wait()


def play(scenes, display, art, marks):
    for index, (seconds, args) in enumerate(scenes):
        command = [display, "--effect", args[0], "--theme", args[1],
                   "--fullscreen", "--file", str(art), *args[2:]]
        child = subprocess.Popen(command)
        try:
            if index == 0:
                time.sleep(0.8)  # Trim the first display's startup from the capture.
                marks["start"] = time.time()
            marks.setdefault("scenes", []).append({"effect": args[0], "at": time.time()})
            try:
                code = child.wait(timeout=seconds)
            except subprocess.TimeoutExpired:
                if index == len(scenes) - 1:
                    marks["end"] = time.time()  # Trim before restoring the terminal.
            else:
                raise RuntimeError(f"{args[0]} exited early (status {code})")
        finally:
            stop(child)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--display", required=True)
    parser.add_argument("--art", type=Path, required=True)
    parser.add_argument("--timeline", type=Path, required=True)
    parser.add_argument("--marks", type=Path, required=True)
    parser.add_argument("--start-file", type=Path, required=True)
    options = parser.parse_args()
    marks = {}

    def interrupted(*_):
        raise KeyboardInterrupt

    signal.signal(signal.SIGTERM, interrupted)
    try:
        scenes = parse_timeline(options.timeline.read_text())
        if not options.art.is_file():
            raise ValueError(f"artwork missing: {options.art}")
        deadline = time.monotonic() + 30
        while not options.start_file.exists():
            if time.monotonic() >= deadline:
                raise RuntimeError("recorder did not start within 30 seconds")
            time.sleep(0.05)
        play(scenes, options.display, options.art, marks)
    except (OSError, ValueError, RuntimeError, KeyboardInterrupt) as error:
        marks["error"] = str(error) or "interrupted"
    temporary = options.marks.with_suffix(".tmp")
    temporary.write_text(json.dumps(marks))
    temporary.replace(options.marks)
    return int("error" in marks)


if __name__ == "__main__":
    raise SystemExit(main())
