from __future__ import annotations

# pylint: disable=protected-access,duplicate-code

import importlib.util
import sys
from pathlib import Path

import pytest


REPO_ROOT = Path(__file__).resolve().parents[1]


def _load_module(name: str, relative_path: str):
    module_path = REPO_ROOT / relative_path
    spec = importlib.util.spec_from_file_location(name, module_path)
    if spec is None or spec.loader is None:
        raise RuntimeError(f"Unable to load module from {module_path}")
    module = importlib.util.module_from_spec(spec)
    sys.modules[name] = module
    spec.loader.exec_module(module)
    return module


def _expect_equal(actual: object, expected: object, message: str) -> None:
    if actual != expected:
        pytest.fail(f"{message}: expected {expected!r}, got {actual!r}")


def test_request_https_json_validates_host_and_decodes_json(monkeypatch: pytest.MonkeyPatch) -> None:
    module = _load_module("security_helpers_under_test", "scripts/security_helpers.py")

    observed: dict[str, object] = {}

    def fake_execute_https_request(**kwargs):
        observed.update(kwargs)
        return 200, "OK", '{"ok": true}', {"x-test": "1"}

    monkeypatch.setattr(module, "_execute_https_request", fake_execute_https_request)

    payload, headers = module.request_https_json(
        "https://api.github.com/repos/owner/repo",
        headers={"Accept": "application/json"},
        allowed_hosts={"api.github.com"},
    )

    _expect_equal(payload, {"ok": True}, "request_https_json should decode the JSON payload")
    _expect_equal(headers, {"x-test": "1"}, "request_https_json should return response headers")
    _expect_equal(observed["host"], "api.github.com", "request_https_json should validate the parsed host")
    _expect_equal(
        observed["request_target"],
        "/repos/owner/repo",
        "request_https_json should send the expected request target",
    )


def test_check_required_checks_api_get_uses_secure_helper(monkeypatch: pytest.MonkeyPatch) -> None:
    module = _load_module("check_required_checks_under_test", "scripts/quality/check_required_checks.py")

    observed: dict[str, object] = {}

    def fake_request_https_json(url: str, **kwargs):
        observed["url"] = url
        observed.update(kwargs)
        return {"ok": True}, {"x-test": "1"}

    monkeypatch.setattr(module, "request_https_json", fake_request_https_json, raising=False)

    payload = module._api_get("owner/repo", "commits/abc/status", "token")

    _expect_equal(payload, {"ok": True}, "_api_get should return the helper payload")
    _expect_equal(
        observed["url"],
        "https://api.github.com/repos/owner/repo/commits/abc/status",
        "_api_get should call the GitHub API URL",
    )
    _expect_equal(observed["allowed_hosts"], {"api.github.com"}, "_api_get should allow only api.github.com")
    _expect_equal(observed["method"], "GET", "_api_get should issue a GET request")


def test_safe_output_path_in_workspace_rejects_escape(tmp_path: Path) -> None:
    module = _load_module("security_helpers_output_paths", "scripts/security_helpers.py")

    inside = module.safe_output_path_in_workspace("reports/out.json", "fallback.json", base=tmp_path)
    _expect_equal(
        inside,
        tmp_path / "reports" / "out.json",
        "safe_output_path_in_workspace should resolve the in-workspace output path",
    )

    with pytest.raises(ValueError, match="escapes workspace root"):
        module.safe_output_path_in_workspace("../outside.json", "fallback.json", base=tmp_path)


def test_safe_input_file_path_in_workspace_requires_existing_workspace_file(tmp_path: Path) -> None:
    module = _load_module("security_helpers_input_paths", "scripts/security_helpers.py")

    coverage_file = tmp_path / "coverage" / "go.xml"
    coverage_file.parent.mkdir(parents=True)
    coverage_file.write_text("<coverage />", encoding="utf-8")

    _expect_equal(
        module.safe_input_file_path_in_workspace("coverage/go.xml", base=tmp_path),
        coverage_file,
        "safe_input_file_path_in_workspace should return the in-workspace file",
    )

    with pytest.raises(ValueError, match="escapes workspace root"):
        module.safe_input_file_path_in_workspace("../outside.xml", base=tmp_path)

    with pytest.raises(ValueError, match="does not exist"):
        module.safe_input_file_path_in_workspace("coverage/missing.xml", base=tmp_path)


def test_assert_coverage_named_path_stays_within_workspace(tmp_path: Path, monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.chdir(tmp_path)
    module = _load_module("assert_coverage_100_under_test", "scripts/quality/assert_coverage_100.py")

    coverage_file = tmp_path / "artifacts" / "coverage.xml"
    coverage_file.parent.mkdir(parents=True)
    coverage_file.write_text("<coverage />", encoding="utf-8")

    name, path = module.parse_named_path("go=artifacts/coverage.xml")
    _expect_equal(name, "go", "parse_named_path should preserve the metric name")
    _expect_equal(path, coverage_file, "parse_named_path should resolve the coverage file inside the workspace")

    with pytest.raises(ValueError, match="escapes workspace root"):
        module.parse_named_path("go=../coverage.xml")
