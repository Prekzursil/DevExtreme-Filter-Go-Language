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

The output also carries branch coverage data: each Go cover region
counts as one ``branch`` for the lines it spans, and ``coveredBranches``
counts regions whose hit count is non-zero.  This supplies SonarCloud's
``new_branch_coverage`` and ``branch_coverage`` metrics, which the
default Go Cover sensor cannot supply since the Go cover format doesn't
distinguish branches natively.

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


# Per-line accumulator: total branches touching the line, branches with hits.
class LineCoverage:
    """Tracks branches-to-cover and covered-branches for a single line."""

    __slots__ = ("branches_to_cover", "covered_branches")

    def __init__(self) -> None:
        """Initialise both counters to zero."""
        self.branches_to_cover = 0
        self.covered_branches = 0

    def add_region(self, hits: int) -> None:
        """Record one Go cover region touching this line."""
        self.branches_to_cover += 1
        if hits > 0:
            self.covered_branches += 1

    @property
    def is_covered(self) -> bool:
        """Return True iff at least one region covering the line was hit."""
        return self.covered_branches > 0


def parse_coverage(
    in_path: Path,
    module: str,
) -> dict[str, dict[int, LineCoverage]]:
    """Parse ``coverage.out`` and return per-file/per-line coverage.

    Each Go cover region is treated as one branch for the lines it
    spans.  This gives Sonar both ``lines`` (covered if any region with
    hits>0 touches the line) and ``branches`` (region count) data.
    """
    by_file: dict[str, dict[int, LineCoverage]] = defaultdict(dict)
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
                bucket = by_file[file_path].setdefault(ln, LineCoverage())
                bucket.add_region(hits)
    return by_file


def emit_xml(
    by_file: dict[str, dict[int, LineCoverage]],
    out_path: Path,
) -> int:
    """Emit Sonar Generic Test Coverage XML and return the file count."""
    out_lines: list[str] = ['<coverage version="1">']
    for file_path in sorted(by_file):
        out_lines.append(f"  <file path={quoteattr(file_path)}>")
        for ln in sorted(by_file[file_path]):
            cov = by_file[file_path][ln]
            covered = "true" if cov.is_covered else "false"
            out_lines.append(
                f'    <lineToCover lineNumber="{ln}" covered="{covered}"'
                f' branchesToCover="{cov.branches_to_cover}"'
                f' coveredBranches="{cov.covered_branches}"/>'
            )
        out_lines.append("  </file>")
    out_lines.append("</coverage>")
    out_lines.append("")  # trailing newline
    out_path.write_text("\n".join(out_lines), encoding="utf-8")
    return len(by_file)


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


def summarise(by_file: dict[str, dict[int, LineCoverage]]) -> tuple[int, int, int, int]:
    """Return (total_lines, covered_lines, total_branches, covered_branches)."""
    total_lines = 0
    covered_lines = 0
    total_branches = 0
    covered_branches = 0
    for line_map in by_file.values():
        for cov in line_map.values():
            total_lines += 1
            if cov.is_covered:
                covered_lines += 1
            total_branches += cov.branches_to_cover
            covered_branches += cov.covered_branches
    return total_lines, covered_lines, total_branches, covered_branches


def main() -> int:
    """CLI entry point."""
    args = parse_args()
    in_path = Path(args.in_path)
    out_path = Path(args.out_path)
    by_file = parse_coverage(in_path, args.module)
    file_count = emit_xml(by_file, out_path)
    total_lines, covered_lines, total_branches, covered_branches = summarise(by_file)
    print(
        f"Wrote {out_path}: {file_count} files, "
        f"lines {covered_lines}/{total_lines}, "
        f"branches {covered_branches}/{total_branches}"
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
