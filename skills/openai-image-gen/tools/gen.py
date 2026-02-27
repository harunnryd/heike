#!/usr/bin/env python3
"""Batch image generator for OpenAI Images API.

Features:
- Deterministic-ish themed prompt synthesis
- Model-aware defaults and basic parameter validation
- Local artifact bundle: images + manifest.json + index.html
"""

from __future__ import annotations

import argparse
import base64
import datetime as dt
import json
import os
import random
import re
import sys
import urllib.error
import urllib.request
from dataclasses import dataclass
from pathlib import Path
from typing import Any

API_URL = "https://api.openai.com/v1/images/generations"

MODEL_DEFAULTS = {
    "gpt-image-1": {"size": "1024x1024", "quality": "high"},
    "gpt-image-1-mini": {"size": "1024x1024", "quality": "medium"},
    "gpt-image-1.5": {"size": "1024x1024", "quality": "high"},
    "dall-e-3": {"size": "1024x1024", "quality": "standard"},
    "dall-e-2": {"size": "1024x1024", "quality": "standard"},
}

GPT_IMAGE_MODELS = {"gpt-image-1", "gpt-image-1-mini", "gpt-image-1.5"}


@dataclass
class JobConfig:
    model: str
    size: str
    quality: str
    background: str
    output_format: str
    style: str
    count: int
    out_dir: Path
    prompt: str


def sanitize_slug(text: str) -> str:
    text = text.strip().lower()
    text = re.sub(r"[^a-z0-9]+", "-", text)
    text = re.sub(r"-{2,}", "-", text).strip("-")
    return text or "image"


def choose_output_dir(explicit: str) -> Path:
    if explicit:
        return Path(explicit).expanduser()

    timestamp = dt.datetime.now().strftime("%Y%m%d-%H%M%S")
    preferred = Path.home() / "Projects" / "tmp"
    root = preferred if preferred.is_dir() else Path("./tmp")
    return root / f"heike-image-gen-{timestamp}"


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Generate images with OpenAI Images API")
    parser.add_argument("--prompt", default="", help="Single prompt. Empty means auto prompt generation.")
    parser.add_argument("--count", type=int, default=6, help="How many images to generate.")
    parser.add_argument("--model", default="gpt-image-1", help="Image model id.")
    parser.add_argument("--size", default="", help="Image size override.")
    parser.add_argument("--quality", default="", help="Quality override.")
    parser.add_argument("--background", default="", help="GPT image models only: transparent, opaque, auto.")
    parser.add_argument("--output-format", default="", help="GPT image models only: png, jpeg, webp.")
    parser.add_argument("--style", default="", help="dall-e-3 style: vivid or natural.")
    parser.add_argument("--out-dir", default="", help="Output directory.")
    parser.add_argument("--input", default="", help="Runtime JSON object input.")
    return parser.parse_args()


def apply_runtime_input(args: argparse.Namespace) -> bool:
    raw = args.input.strip()
    if not raw:
        return False

    try:
        payload = json.loads(raw)
    except json.JSONDecodeError as err:
        raise ValueError(f"invalid --input JSON: {err}") from err

    if not isinstance(payload, dict):
        raise ValueError("--input must be a JSON object")

    runtime_count_set = False

    if "prompt" in payload and not args.prompt:
        args.prompt = str(payload["prompt"])
    if "model" in payload and args.model == "gpt-image-1":
        args.model = str(payload["model"])
    if "size" in payload and not args.size:
        args.size = str(payload["size"])
    if "quality" in payload and not args.quality:
        args.quality = str(payload["quality"])
    if "background" in payload and not args.background:
        args.background = str(payload["background"])
    if "output_format" in payload and not args.output_format:
        args.output_format = str(payload["output_format"])
    if "style" in payload and not args.style:
        args.style = str(payload["style"])
    if "out_dir" in payload and not args.out_dir:
        args.out_dir = str(payload["out_dir"])
    if "count" in payload:
        try:
            args.count = int(payload["count"])
        except (TypeError, ValueError) as err:
            raise ValueError("count must be an integer") from err
        runtime_count_set = True

    # Runtime calls should default to low-volume generation when count is omitted.
    if not runtime_count_set and args.count == 6:
        args.count = 1

    return True


def build_job_config(args: argparse.Namespace) -> JobConfig:
    model = args.model.strip()
    defaults = MODEL_DEFAULTS.get(model, MODEL_DEFAULTS["gpt-image-1"])

    count = max(1, args.count)
    if model == "dall-e-3" and count > 1:
        print("warning: dall-e-3 supports one image per request; forcing count=1", file=sys.stderr)
        count = 1

    return JobConfig(
        model=model,
        size=args.size.strip() or defaults["size"],
        quality=args.quality.strip() or defaults["quality"],
        background=args.background.strip(),
        output_format=args.output_format.strip(),
        style=args.style.strip(),
        count=count,
        out_dir=choose_output_dir(args.out_dir),
        prompt=args.prompt.strip(),
    )


def build_prompt_set(prompt: str, count: int) -> list[str]:
    if prompt:
        return [prompt] * count

    themes = [
        "retro-futuristic transit hub",
        "quiet midnight bookstore",
        "minimal product hero shot",
        "stormy coastal observatory",
        "sci-fi greenhouse interior",
        "street food scene in neon rain",
        "craft workshop with warm tungsten light",
    ]
    render_styles = [
        "cinematic still",
        "editorial photo",
        "isometric illustration",
        "matte painting",
        "photoreal studio render",
    ]
    camera_mood = [
        "wide-angle composition",
        "shallow depth of field",
        "high contrast",
        "soft atmospheric haze",
        "clean geometric framing",
    ]

    prompts: list[str] = []
    for _ in range(count):
        prompts.append(
            f"{random.choice(render_styles)} of {random.choice(themes)}, {random.choice(camera_mood)}"
        )
    return prompts


def request_image(api_key: str, cfg: JobConfig, prompt: str) -> dict[str, Any]:
    payload: dict[str, Any] = {
        "model": cfg.model,
        "prompt": prompt,
        "size": cfg.size,
        "n": 1,
    }

    if cfg.model != "dall-e-2":
        payload["quality"] = cfg.quality

    if cfg.model in GPT_IMAGE_MODELS:
        if cfg.background:
            payload["background"] = cfg.background
        if cfg.output_format:
            payload["output_format"] = cfg.output_format

    if cfg.model == "dall-e-3" and cfg.style:
        payload["style"] = cfg.style

    req = urllib.request.Request(
        API_URL,
        method="POST",
        headers={
            "Authorization": f"Bearer {api_key}",
            "Content-Type": "application/json",
        },
        data=json.dumps(payload).encode("utf-8"),
    )

    try:
        with urllib.request.urlopen(req, timeout=300) as response:
            return json.loads(response.read().decode("utf-8"))
    except urllib.error.HTTPError as err:
        details = err.read().decode("utf-8", errors="replace")
        raise RuntimeError(f"images api http {err.code}: {details}") from err


def infer_extension(cfg: JobConfig) -> str:
    if cfg.model in GPT_IMAGE_MODELS and cfg.output_format:
        return cfg.output_format
    return "png"


def write_image(data: dict[str, Any], target: Path) -> None:
    b64 = data.get("b64_json")
    url = data.get("url")

    if b64:
        target.write_bytes(base64.b64decode(b64))
        return

    if not url:
        raise RuntimeError("response does not include b64_json or url")

    try:
        urllib.request.urlretrieve(url, target)
    except urllib.error.URLError as err:
        raise RuntimeError(f"failed downloading {url}: {err}") from err


def render_gallery(out_dir: Path, records: list[dict[str, str]]) -> None:
    cards = []
    for item in records:
        cards.append(
            "\n".join(
                [
                    "<article class=\"card\">",
                    f"  <img src=\"{item['file']}\" alt=\"generated\" loading=\"lazy\" />",
                    f"  <p>{item['prompt']}</p>",
                    "</article>",
                ]
            )
        )

    html = f"""<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>heike openai-image-gen</title>
  <style>
    body {{ font-family: ui-sans-serif, system-ui, -apple-system, Segoe UI, sans-serif; margin: 24px; background: #0b1320; color: #e8edf7; }}
    h1 {{ margin: 0 0 10px; font-size: 20px; }}
    .meta {{ opacity: 0.85; margin-bottom: 20px; }}
    .grid {{ display: grid; gap: 14px; grid-template-columns: repeat(auto-fill, minmax(220px, 1fr)); }}
    .card {{ background: #131d2e; border: 1px solid #243049; border-radius: 12px; padding: 10px; }}
    .card img {{ width: 100%; border-radius: 8px; display: block; }}
    .card p {{ font-size: 12px; line-height: 1.4; color: #b9c5dd; }}
    code {{ color: #9dd0ff; }}
  </style>
</head>
<body>
  <h1>heike openai-image-gen</h1>
  <p class="meta">Output directory: <code>{out_dir.as_posix()}</code></p>
  <section class="grid">
    {''.join(cards)}
  </section>
</body>
</html>
"""

    (out_dir / "index.html").write_text(html, encoding="utf-8")


def main() -> int:
    args = parse_args()
    try:
        runtime_mode = apply_runtime_input(args)
    except ValueError as err:
        print(f"error: {err}", file=sys.stderr)
        return 2

    cfg = build_job_config(args)

    api_key = (os.getenv("OPENAI_API_KEY") or "").strip()
    if not api_key:
        print("error: OPENAI_API_KEY is required", file=sys.stderr)
        return 2

    cfg.out_dir.mkdir(parents=True, exist_ok=True)
    prompts = build_prompt_set(cfg.prompt, cfg.count)
    ext = infer_extension(cfg)

    records: list[dict[str, str]] = []
    for idx, prompt in enumerate(prompts, start=1):
        if runtime_mode:
            print(f"[{idx}/{len(prompts)}] {prompt}", file=sys.stderr)
        else:
            print(f"[{idx}/{len(prompts)}] {prompt}")
        result = request_image(api_key, cfg, prompt)
        data = (result.get("data") or [{}])[0]

        filename = f"{idx:03d}-{sanitize_slug(prompt)[:48]}.{ext}"
        target = cfg.out_dir / filename
        write_image(data, target)

        records.append(
            {
                "file": filename,
                "prompt": prompt,
                "model": cfg.model,
                "size": cfg.size,
            }
        )

    (cfg.out_dir / "manifest.json").write_text(json.dumps(records, indent=2), encoding="utf-8")
    render_gallery(cfg.out_dir, records)

    manifest_path = (cfg.out_dir / "manifest.json").as_posix()
    gallery_path = (cfg.out_dir / "index.html").as_posix()
    if runtime_mode:
        print(
            json.dumps(
                {
                    "status": "ok",
                    "tool": "openai_image_gen",
                    "manifest": manifest_path,
                    "gallery": gallery_path,
                    "out_dir": cfg.out_dir.as_posix(),
                    "count": len(records),
                    "model": cfg.model,
                }
            )
        )
    else:
        print(f"\nmanifest: {manifest_path}")
        print(f"gallery : {gallery_path}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
