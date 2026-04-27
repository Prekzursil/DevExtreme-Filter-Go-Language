#!/usr/bin/env python3
"""Convert Go cover format to Sonar Generic Test Coverage XML.

The SonarCloud Go Cover sensor has been silently dropping all hits in our
analysis (39ms parse, 0% applied) despite ``coverage.out`` containing 604
lines of valid Go cover data with 100% statement coverage. The likely
cause is module-prefix path resolution returning ``null`` for every
indexed-file lookup.

Sonar Generic Test Coverage XML uses ``path`` attributes that map 1:1
against indexed source files, so emitting it directly bypasses the path
resolution problem.

Format reference:
    https://docs.sonarsource.com/sonarqube/latest/analyzing-source-code/test-coverage/generic-test-data/

Usage:
    python gocover_to_sonar_generic.py \
        --in coverage.out \
        --module transaction-filter-backend \
        --out coverage-sonar.xml
"""
from __future__ import annotations

import argparse
import re
from collections import defaultdict
from pathlib import Path
from xml.sax.saxutils import quoteattr

# A Go cover region:
#   <import-path>/<file>:<startLine>.<startCol>,<endLine>.<endCol> <numStatements> <count>
GO_COVER_LINE = re.compile(
    r"^(?P<path>[^:]+):"
    r"(?P<start_line>\d+)\.(?P<start_col>\d+),"
    r"(?P<end_line>\d+)\.(?P<end_col>\d+) "
    r"(?P<num_stmts>\d+) "
    r"(?P<hits>\d+)$"
)


def strip_module_prefix(path: str, module: str) -> str:
    """Strip the Go module prefix so the result is repo-relative."""
    if module and path == module:
        return "."
    if module and path.startswith(module + "/"):
        return path[len(module) + 1 :]
    return path


def parse_coverage(in_path: Path, module: str) -> dict[str, dict[int, int]]:
    """Parse ``coverage.out`` and return ``{file: {line: max_hits}}``.

    A region from start_line..end_line covers every line in that range
    (inclusive).  We take the *max* hits seen per line so that overlapping
    regions don't downgrade a covered line to uncovered.
    """
    lines_by_file: dict[str, dict[int, int]] = defaultdict(dict)
    with in_path.open("r", encoding="utf-8") as fh:
        for raw in fh:
            line = raw.strip()
            if not line or line.startswith("mode:"):
                continue
            match = GO_COVER_LINE.match(line)
            if not match:
                continue
            file_path = strip_module_prefix(match.group("path"), module)
            start = int(match.group("start_line"))
            end = int(match.group("end_line"))
            hits = int(match.group("hits"))
            for ln in range(start, end + 1):
                prior = lines_by_file[file_path].get(ln, 0)
                if hits > prior:
                    lines_by_file[file_path][ln] = hits
    return lines_by_file


def emit_xml(lines_by_file: dict[str, dict[int, int]], out_path: Path) -> int:
    """Emit Sonar Generic Test Coverage XML and return the file count."""
    out_lines: list[str] = ['<coverage version="1">']
    for file_path in sorted(lines_by_file):
        out_lines.append(f"  <file path={quoteattr(file_path)}>")
        for ln in sorted(lines_by_file[file_path]):
            hits = lines_by_file[file_path][ln]
            covered = "true" if hits > 0 else "false"
            out_lines.append(
                f'    <lineToCover lineNumber="{ln}" covered="{covered}"/>'
            )
        out_lines.append("  </file>")
    out_lines.append("</coverage>")
    out_lines.append("")  # trailing newline
    out_path.write_text("\n".join(out_lines), encoding="utf-8")
    return len(lines_by_file)


def parse_args() -> argparse.Namespace:
    """Parse CLI arguments."""
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "--in",
        dest="in_path",
        required=True,
        help="Input Go cover profile (coverage.out)",
    )
    parser.add_argument(
        "--module",
        required=True,
        help="Go module name to strip from import paths (from go.mod)",
    )
    parser.add_argument(
        "--out",
        dest="out_path",
        required=True,
        help="Output Sonar Generic Test Coverage XML path",
    )
    return parser.parse_args()


def main() -> int:
    """CLI entry point."""
    args = parse_args()
    in_path = Path(args.in_path)
    out_path = Path(args.out_path)
    lines_by_file = parse_coverage(in_path, args.module)
    file_count = emit_xml(lines_by_file, out_path)
    total_lines = sum(len(v) for v in lines_by_file.values())
    covered_lines = sum(
        sum(1 for h in v.values() if h > 0) for v in lines_by_file.values()
    )
    print(
        f"Wrote {out_path}: {file_count} files, "
        f"{total_lines} lines, {covered_lines} covered"
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
