#!/usr/bin/env python3
import argparse
import datetime as dt
import json
import os
import sys
from pathlib import Path


DEFAULT_MAX_DEPTH = 6
DEFAULT_TOP_EXTENSIONS = 10
DEFAULT_MAX_FILES = 5000
MAX_TEXT_FILE_BYTES = 2 * 1024 * 1024


def workspace_root() -> Path:
    return Path(__file__).resolve().parents[3]


def parse_runtime_input(raw: str) -> dict:
    if not raw.strip():
        return {}
    data = json.loads(raw)
    if not isinstance(data, dict):
        raise ValueError("input must be a JSON object")
    return data


def to_int(value, default: int, minimum: int, maximum: int) -> int:
    if value is None:
        return default
    try:
        parsed = int(value)
    except (TypeError, ValueError):
        return default
    if parsed < minimum:
        return minimum
    if parsed > maximum:
        return maximum
    return parsed


def to_bool(value, default: bool) -> bool:
    if value is None:
        return default
    if isinstance(value, bool):
        return value
    if isinstance(value, str):
        lowered = value.strip().lower()
        if lowered in {"1", "true", "yes", "y"}:
            return True
        if lowered in {"0", "false", "no", "n"}:
            return False
    return default


def normalize_ext(path: Path) -> str:
    suffix = path.suffix.lower().lstrip(".")
    if suffix:
        return suffix
    return "no_ext"


def line_count(path: Path) -> int:
    count = 0
    with path.open("r", encoding="utf-8", errors="ignore") as handle:
        for _ in handle:
            count += 1
    return count


def is_binary_or_too_large(path: Path) -> bool:
    try:
        size = path.stat().st_size
    except OSError:
        return True
    if size > MAX_TEXT_FILE_BYTES:
        return True

    try:
        with path.open("rb") as handle:
            sample = handle.read(2048)
    except OSError:
        return True
    return b"\x00" in sample


def should_skip_name(name: str, include_hidden: bool) -> bool:
    if include_hidden:
        return False
    return name.startswith(".")


def within_depth(scan_root: Path, path: Path, max_depth: int) -> bool:
    rel = path.relative_to(scan_root)
    return len(rel.parts) <= max_depth


def collect_stats(scan_root: Path, max_depth: int, include_hidden: bool, max_files: int) -> dict:
    ext_stats = {}
    scanned_files = 0
    scanned_dirs = 0
    skipped_files = 0
    truncated = False
    total_lines = 0

    for current_root, dirs, files in os.walk(scan_root):
        root_path = Path(current_root)
        if not within_depth(scan_root, root_path, max_depth):
            dirs[:] = []
            continue

        dirs[:] = [d for d in dirs if not should_skip_name(d, include_hidden)]
        files = [f for f in files if not should_skip_name(f, include_hidden)]
        scanned_dirs += 1

        for filename in sorted(files):
            file_path = root_path / filename
            if not file_path.is_file() or file_path.is_symlink():
                skipped_files += 1
                continue
            if is_binary_or_too_large(file_path):
                skipped_files += 1
                continue

            scanned_files += 1
            if scanned_files > max_files:
                truncated = True
                return {
                    "ext_stats": ext_stats,
                    "scanned_files": max_files,
                    "scanned_dirs": scanned_dirs,
                    "skipped_files": skipped_files,
                    "total_lines": total_lines,
                    "truncated": truncated,
                }

            ext = normalize_ext(file_path)
            lines = line_count(file_path)
            total_lines += lines

            bucket = ext_stats.setdefault(ext, {"files": 0, "lines": 0})
            bucket["files"] += 1
            bucket["lines"] += lines

    return {
        "ext_stats": ext_stats,
        "scanned_files": scanned_files,
        "scanned_dirs": scanned_dirs,
        "skipped_files": skipped_files,
        "total_lines": total_lines,
        "truncated": truncated,
    }


def build_output(scan_root: Path, stats: dict, top_extensions: int) -> dict:
    groups = []
    for ext, values in stats["ext_stats"].items():
        groups.append(
            {
                "ext": ext,
                "files": values["files"],
                "lines": values["lines"],
            }
        )

    groups.sort(key=lambda item: (-item["files"], -item["lines"], item["ext"]))
    groups = groups[:top_extensions]

    return {
        "status": "ok",
        "tool": "codebase_stats",
        "root": scan_root.as_posix(),
        "scanned_files": stats["scanned_files"],
        "scanned_dirs": stats["scanned_dirs"],
        "skipped_files": stats["skipped_files"],
        "total_lines": stats["total_lines"],
        "top_extensions": groups,
        "truncated": stats["truncated"],
        "generated_at": dt.datetime.utcnow().replace(microsecond=0).isoformat() + "Z",
    }


def main() -> int:
    parser = argparse.ArgumentParser(description="Compute deterministic codebase statistics.")
    parser.add_argument("--input", default="", help="Runtime JSON input object.")
    args = parser.parse_args()

    try:
        payload = parse_runtime_input(args.input)
    except Exception as err:  # pragma: no cover
        print(json.dumps({"status": "error", "error": f"invalid input: {err}"}))
        return 1

    root_value = str(payload.get("root", "")).strip()
    base_root = workspace_root()
    scan_root = (base_root / root_value).resolve() if root_value else base_root.resolve()
    if not scan_root.exists() or not scan_root.is_dir():
        print(json.dumps({"status": "error", "error": f"root not found: {scan_root.as_posix()}"}))
        return 1

    try:
        scan_root.relative_to(base_root.resolve())
    except ValueError:
        print(json.dumps({"status": "error", "error": "root must stay within workspace"}))
        return 1

    max_depth = to_int(payload.get("max_depth"), DEFAULT_MAX_DEPTH, 1, 32)
    top_extensions = to_int(payload.get("top_extensions"), DEFAULT_TOP_EXTENSIONS, 1, 50)
    max_files = to_int(payload.get("max_files"), DEFAULT_MAX_FILES, 1, 20000)
    include_hidden = to_bool(payload.get("include_hidden"), False)

    stats = collect_stats(scan_root, max_depth, include_hidden, max_files)
    output = build_output(scan_root, stats, top_extensions)
    print(json.dumps(output, separators=(",", ":")))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
