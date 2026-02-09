#!/usr/bin/env python3
"""Generate a dot-only MP4 animation of OpcodeOracle disassembly progress."""

from __future__ import annotations

import argparse
import glob
import math
import os
import re
import shutil
import subprocess
import sys
import tempfile
from dataclasses import dataclass
from pathlib import Path

from PIL import Image, ImageDraw, ImageFilter

ADDRESS_RE = re.compile(
    r"(?<![0-9A-Fa-f])\$(?:[0-9A-Fa-f]{2}|[0-9A-Fa-f]{4})(?![0-9A-Fa-f])"
)
OPCODE_RE = re.compile(r"^ {4}([A-Z]{3}) ")


def fail(message: str) -> None:
    print(f"error: {message}", file=sys.stderr)
    sys.exit(1)


def parse_hex_color(value: str) -> tuple[int, int, int]:
    value = value.strip()
    if not re.fullmatch(r"#[0-9a-fA-F]{6}", value):
        fail(f"invalid color '{value}' (expected #RRGGBB)")
    return tuple(int(value[i : i + 2], 16) for i in (1, 3, 5))


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Generate a dot-only progress animation from archived state files."
    )
    parser.add_argument(
        "--input-glob",
        default="testdata/archive/*.opcodeoracle.json",
        help="Glob pattern for input state files.",
    )
    parser.add_argument(
        "--opcodeoracle-bin",
        default="./opcodeoracle",
        help="Path to opcodeoracle executable.",
    )
    parser.add_argument(
        "--output",
        default="testdata/archive/disasm_progress_960x540.mp4",
        help="Output MP4 file path.",
    )
    parser.add_argument("--width", type=int, default=960, help="Output video width.")
    parser.add_argument("--height", type=int, default=540, help="Output video height.")
    parser.add_argument(
        "--seconds", type=float, default=12.0, help="Target video duration in seconds."
    )
    parser.add_argument("--fps", type=int, default=30, help="Output frames per second.")
    parser.add_argument(
        "--max-frames",
        type=int,
        default=0,
        help="Optional hard cap for frame count (0 means auto from seconds*fps).",
    )
    parser.add_argument(
        "--columns",
        type=int,
        default=0,
        help="Fixed number of line columns. 0 = auto-select.",
    )
    parser.add_argument(
        "--min-columns",
        type=int,
        default=5,
        help="Minimum auto-selected columns.",
    )
    parser.add_argument(
        "--max-columns",
        type=int,
        default=8,
        help="Maximum auto-selected columns.",
    )
    parser.add_argument(
        "--column-gap",
        type=int,
        default=10,
        help="Pixel gap between columns.",
    )
    parser.add_argument(
        "--bg-color", default="#0D0A07", help="Background color as hex, e.g. #0D0A07."
    )
    parser.add_argument(
        "--dot-color",
        default="#FFB347",
        help="Deprecated alias for --address-color.",
    )
    parser.add_argument(
        "--address-color",
        default=None,
        help="Color for $xx/$xxxx tokens as hex (default: same as --dot-color).",
    )
    parser.add_argument(
        "--comment-color",
        default="#5A5A5A",
        help="Color for comment text (; ...) as hex.",
    )
    parser.add_argument(
        "--text-color",
        default="#C8C8C8",
        help="Color for all other non-whitespace text as hex.",
    )
    parser.add_argument(
        "--opcode-color",
        default="#8FD3FF",
        help="Color for leading 3-letter opcodes (pattern: four spaces + AAA + space).",
    )
    parser.add_argument(
        "--dot-radius",
        type=int,
        default=1,
        help="Dot radius in pixels (1 = single pixel dots).",
    )
    parser.add_argument(
        "--ffmpeg-crf", type=int, default=34, help="Initial x264 CRF value."
    )
    parser.add_argument(
        "--ffmpeg-preset", default="slow", help="x264 preset (e.g. slow, medium, fast)."
    )
    parser.add_argument(
        "--keep-temp-frames",
        action="store_true",
        help="Keep temporary PNG frames directory.",
    )
    return parser.parse_args()


@dataclass
class LineLayout:
    cols: int
    rows_per_col: int
    start_x: int
    start_y: int
    column_width: float
    row_height: float
    char_width: float
    column_gap: int
    max_line_len: int


def find_inputs(pattern: str) -> list[str]:
    files = sorted(glob.glob(pattern))
    files = [f for f in files if f.endswith(".opcodeoracle.json")]
    if not files:
        fail(f"no input files found for glob: {pattern}")
    if len(files) < 2:
        fail("at least two input files are required to animate progress")
    return files


def sampled_indices(total: int, wanted: int) -> list[int]:
    if wanted <= 0:
        fail("frame count must be positive")
    if wanted >= total:
        return list(range(total))
    if wanted == 1:
        return [total - 1]
    idxs: list[int] = []
    for i in range(wanted):
        pos = int(round(i * (total - 1) / (wanted - 1)))
        if not idxs or pos != idxs[-1]:
            idxs.append(pos)
    return idxs


def run_disasm(opcodeoracle_bin: str, state_file: str) -> str:
    cmd = [opcodeoracle_bin, "disasm", state_file]
    try:
        proc = subprocess.run(cmd, check=True, capture_output=True, text=True)
    except FileNotFoundError:
        fail(f"opcodeoracle binary not found: {opcodeoracle_bin}")
    except subprocess.CalledProcessError as exc:
        fail(
            "disasm failed for "
            f"{state_file}\n"
            f"command: {' '.join(cmd)}\n"
            f"stderr:\n{exc.stderr.strip()}"
        )
    return proc.stdout.replace("\r\n", "\n").replace("\r", "\n")


def choose_columns(
    max_lines: int,
    max_line_len: int,
    drawable_width: int,
    drawable_height: int,
    min_cols: int,
    max_cols: int,
    col_gap: int,
    fixed_cols: int,
) -> int:
    if fixed_cols > 0:
        return fixed_cols

    best_cols = min_cols
    best_score = float("-inf")
    for cols in range(min_cols, max_cols + 1):
        rows = math.ceil(max_lines / cols)
        total_gap = col_gap * (cols - 1)
        avail_w = drawable_width - total_gap
        if avail_w <= cols:
            continue
        col_w = avail_w / cols
        row_h = drawable_height / rows
        char_w = col_w / max_line_len
        # Balanced objective to avoid jittery over-compression.
        score = min(row_h, char_w * 2.2)
        if score > best_score:
            best_score = score
            best_cols = cols
    return best_cols


def build_layout(
    width: int,
    height: int,
    max_lines: int,
    max_line_len: int,
    min_cols: int,
    max_cols: int,
    col_gap: int,
    fixed_cols: int,
) -> LineLayout:
    margin_x = max(16, int(width * 0.03))
    margin_top = max(12, int(height * 0.03))
    margin_bottom = max(32, int(height * 0.08))
    drawable_w = width - 2 * margin_x
    drawable_h = height - margin_top - margin_bottom
    if drawable_w <= 0 or drawable_h <= 0:
        fail("invalid width/height: drawable area is not positive")

    cols = choose_columns(
        max_lines=max_lines,
        max_line_len=max_line_len,
        drawable_width=drawable_w,
        drawable_height=drawable_h,
        min_cols=min_cols,
        max_cols=max_cols,
        col_gap=col_gap,
        fixed_cols=fixed_cols,
    )
    rows = math.ceil(max_lines / cols)
    avail_w = drawable_w - col_gap * (cols - 1)
    if avail_w <= 0:
        fail("column-gap too large for drawable width")

    col_w = avail_w / cols
    row_h = drawable_h / rows
    char_w = col_w / max_line_len
    return LineLayout(
        cols=cols,
        rows_per_col=rows,
        start_x=margin_x,
        start_y=margin_top,
        column_width=col_w,
        row_height=row_h,
        char_width=char_w,
        column_gap=col_gap,
        max_line_len=max_line_len,
    )


def to_lines(text: str) -> list[str]:
    return text.splitlines()


def _address_char_positions(code_part: str) -> set[int]:
    addr_positions: set[int] = set()
    for match in ADDRESS_RE.finditer(code_part):
        start, end = match.span()
        for i in range(start, end):
            if not code_part[i].isspace():
                addr_positions.add(i)
    return addr_positions


def collect_points_by_kind(
    lines: list[str],
    layout: LineLayout,
    width: int,
    height: int,
) -> dict[str, list[tuple[int, int]]]:
    by_kind: dict[str, list[tuple[int, int]]] = {
        "address": [],
        "comment": [],
        "opcode": [],
        "text": [],
    }
    max_lines = layout.cols * layout.rows_per_col
    for line_idx, line in enumerate(lines[:max_lines]):
        col = line_idx // layout.rows_per_col
        row = line_idx % layout.rows_per_col
        if col >= layout.cols:
            break
        col_start = int(layout.start_x + col * (layout.column_width + layout.column_gap))
        y = int(layout.start_y + row * layout.row_height)
        if y < 0 or y >= height:
            continue

        trimmed = line[: layout.max_line_len]
        comment_idx = trimmed.find(";")
        if comment_idx >= 0:
            code_part = trimmed[:comment_idx]
            comment_part = trimmed[comment_idx:]
        else:
            code_part = trimmed
            comment_part = ""
        addr_positions = _address_char_positions(code_part)
        opcode_positions: set[int] = set()
        opcode_match = OPCODE_RE.match(code_part)
        if opcode_match:
            start, end = opcode_match.span(1)
            opcode_positions.update(range(start, end))

        last_x = {"address": -1, "comment": -1, "opcode": -1, "text": -1}
        for char_idx, ch in enumerate(trimmed):
            if ch.isspace():
                continue
            x = int(col_start + char_idx * layout.char_width)
            if comment_idx >= 0 and char_idx >= comment_idx:
                kind = "comment"
            elif char_idx in opcode_positions:
                kind = "opcode"
            elif char_idx in addr_positions:
                kind = "address"
            else:
                kind = "text"
            if x == last_x[kind]:
                continue
            if 0 <= x < width:
                by_kind[kind].append((x, y))
                last_x[kind] = x
    return by_kind


def paint_frame(
    lines: list[str],
    layout: LineLayout,
    width: int,
    height: int,
    bg_rgb: tuple[int, int, int],
    address_rgb: tuple[int, int, int],
    comment_rgb: tuple[int, int, int],
    opcode_rgb: tuple[int, int, int],
    text_rgb: tuple[int, int, int],
    dot_radius: int,
    progress: float,
) -> Image.Image:
    frame = Image.new("RGB", (width, height), bg_rgb)
    points_by_kind = collect_points_by_kind(
        lines=lines,
        layout=layout,
        width=width,
        height=height,
    )

    for kind, color in (
        ("text", text_rgb),
        ("comment", comment_rgb),
        ("opcode", opcode_rgb),
        ("address", address_rgb),
    ):
        points = points_by_kind[kind]
        if not points:
            continue
        mask = Image.new("L", (width, height), 0)
        dmask = ImageDraw.Draw(mask)
        dmask.point(points, fill=255)
        if dot_radius > 1:
            mask = mask.filter(ImageFilter.MaxFilter(size=dot_radius * 2 + 1))
        dots = Image.new("RGB", (width, height), color)
        frame.paste(dots, (0, 0), mask=mask)

    draw = ImageDraw.Draw(frame)
    bar_w = int(width * 0.84)
    bar_h = max(4, height // 120)
    bar_x = (width - bar_w) // 2
    bar_y = height - max(16, height // 30)
    draw.rectangle([bar_x, bar_y - bar_h, bar_x + bar_w, bar_y], fill=(35, 30, 24))
    fill_w = int(bar_w * max(0.0, min(1.0, progress)))
    draw.rectangle([bar_x, bar_y - bar_h, bar_x + fill_w, bar_y], fill=(170, 120, 60))
    return frame


def encode_mp4(
    frame_pattern: str,
    fps: int,
    output_path: str,
    crf: int,
    preset: str,
) -> None:
    ffmpeg_bin = shutil.which("ffmpeg")
    if ffmpeg_bin is None:
        fail("ffmpeg not found in PATH")
    cmd = [
        ffmpeg_bin,
        "-y",
        "-loglevel",
        "error",
        "-framerate",
        str(fps),
        "-i",
        frame_pattern,
        "-c:v",
        "libx264",
        "-pix_fmt",
        "yuv420p",
        "-crf",
        str(crf),
        "-preset",
        preset,
        "-movflags",
        "+faststart",
        output_path,
    ]
    try:
        subprocess.run(cmd, check=True)
    except subprocess.CalledProcessError:
        fail(f"ffmpeg encoding failed for output: {output_path}")


def main() -> None:
    args = parse_args()
    if args.width <= 0 or args.height <= 0:
        fail("width/height must be positive integers")
    if args.seconds <= 0:
        fail("seconds must be positive")
    if args.fps <= 0:
        fail("fps must be positive")
    if args.dot_radius < 0:
        fail("dot-radius must be zero or positive")
    if args.columns < 0:
        fail("columns must be >= 0")
    if args.min_columns <= 0 or args.max_columns <= 0:
        fail("min/max columns must be positive")
    if args.min_columns > args.max_columns:
        fail("min-columns must be <= max-columns")
    if args.column_gap < 0:
        fail("column-gap must be >= 0")
    if args.ffmpeg_crf < 0 or args.ffmpeg_crf > 51:
        fail("ffmpeg-crf must be in range [0, 51]")

    bg_rgb = parse_hex_color(args.bg_color)
    address_color_arg = args.address_color if args.address_color else args.dot_color
    address_rgb = parse_hex_color(address_color_arg)
    comment_rgb = parse_hex_color(args.comment_color)
    opcode_rgb = parse_hex_color(args.opcode_color)
    text_rgb = parse_hex_color(args.text_color)
    files = find_inputs(args.input_glob)

    auto_frames = max(2, int(round(args.seconds * args.fps)))
    wanted_frames = args.max_frames if args.max_frames > 0 else auto_frames
    indices = sampled_indices(len(files), wanted_frames)
    selected = [files[i] for i in indices]

    all_lines: list[list[str]] = []
    for idx, path in enumerate(selected, start=1):
        print(f"[{idx}/{len(selected)}] disasm {Path(path).name}", file=sys.stderr)
        all_lines.append(to_lines(run_disasm(args.opcodeoracle_bin, path)))

    max_lines = max(len(lines) for lines in all_lines)
    max_line_len = max(
        max((len(line) for line in lines), default=1) for lines in all_lines
    )
    layout = build_layout(
        width=args.width,
        height=args.height,
        max_lines=max_lines,
        max_line_len=max_line_len,
        min_cols=args.min_columns,
        max_cols=args.max_columns,
        col_gap=args.column_gap,
        fixed_cols=args.columns,
    )

    output_path = Path(args.output)
    output_path.parent.mkdir(parents=True, exist_ok=True)

    if args.keep_temp_frames:
        tmp_dir = tempfile.mkdtemp(prefix="disasm_progress_frames_")
    else:
        tmp_ctx = tempfile.TemporaryDirectory(prefix="disasm_progress_frames_")
        tmp_dir = tmp_ctx.name

    for i, lines in enumerate(all_lines):
        progress = 0.0 if len(all_lines) <= 1 else i / (len(all_lines) - 1)
        frame = paint_frame(
            lines=lines,
            layout=layout,
            width=args.width,
            height=args.height,
            bg_rgb=bg_rgb,
            address_rgb=address_rgb,
            comment_rgb=comment_rgb,
            opcode_rgb=opcode_rgb,
            text_rgb=text_rgb,
            dot_radius=args.dot_radius,
            progress=progress,
        )
        frame.save(os.path.join(tmp_dir, f"frame_{i:05d}.png"), optimize=True)

    frame_pattern = os.path.join(tmp_dir, "frame_%05d.png")
    encode_mp4(
        frame_pattern=frame_pattern,
        fps=args.fps,
        output_path=str(output_path),
        crf=args.ffmpeg_crf,
        preset=args.ffmpeg_preset,
    )

    size_bytes = output_path.stat().st_size
    if size_bytes > 3 * 1024 * 1024:
        encode_mp4(
            frame_pattern=frame_pattern,
            fps=args.fps,
            output_path=str(output_path),
            crf=min(51, args.ffmpeg_crf + 3),
            preset=args.ffmpeg_preset,
        )
        size_bytes = output_path.stat().st_size

    if not args.keep_temp_frames:
        tmp_ctx.cleanup()

    print(f"output: {output_path}")
    print(f"frames: {len(all_lines)} @ {args.fps} fps")
    print(f"duration: {len(all_lines) / args.fps:.2f}s")
    print(
        "layout: "
        f"{layout.cols} columns, {layout.rows_per_col} rows/column, "
        f"max_line_len={layout.max_line_len}"
    )
    print(
        "colors: "
        f"address={address_color_arg}, comment={args.comment_color}, "
        f"opcode={args.opcode_color}, text={args.text_color}"
    )
    print(f"size: {size_bytes / (1024 * 1024):.2f} MB")
    if args.keep_temp_frames:
        print(f"frames_dir: {tmp_dir}")


if __name__ == "__main__":
    main()
