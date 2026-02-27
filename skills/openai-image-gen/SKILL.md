---
name: "openai-image-gen"
description: "Generate image batches with OpenAI Images API, then produce a local HTML preview page and manifest file."
tags:
  - image
  - generation
  - openai
  - batch
  - preview
tools:
  - "exec_command"
metadata:
  heike:
    icon: "🖼️"
    category: "creative"
    kind: "runtime"
    requires:
      bins:
        - "python3"
      env:
        - "OPENAI_API_KEY"
    outputs:
      - "manifest.json"
      - "index.html"
---

# OpenAI Image Gen

Use this skill when you need fast visual ideation with repeatable batch generation.

## Preconditions

- `python3` is available.
- `OPENAI_API_KEY` is set.

## Run

```bash
openai_image_gen { "count": 4, "prompt": "cinematic portrait of a robot gardener" }
```

## Typical usage

```bash
python3 {baseDir}/tools/gen.py --prompt "cinematic portrait of a robot gardener" --count 4
python3 {baseDir}/tools/gen.py --model gpt-image-1 --size 1536x1024 --quality high
python3 {baseDir}/tools/gen.py --model dall-e-3 --style vivid --count 1
python3 {baseDir}/tools/gen.py --out-dir ./out/concepts --output-format webp
```

## Outputs

- Generated image files (`png` / `jpeg` / `webp`)
- `manifest.json` (prompt, model, size, and file mapping)
- `index.html` preview gallery

## Agent behavior

1. Confirm required environment (`OPENAI_API_KEY`) before execution.
2. Prefer batch generation for exploration, single prompt for targeted revisions.
3. Return final output directory and gallery path in the response.
