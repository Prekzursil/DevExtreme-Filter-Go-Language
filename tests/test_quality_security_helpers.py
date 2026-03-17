from __future__ import annotations

import importlib.util
from pathlib import Path

import pytest


REPO_ROOT = Path(__file__).resolve().parents[1]


def _load_module(name: str, relative_path: str):
    module_path = REPO_ROOT / relative_path
    spec = importlib.util.spec_from_file_location(name, module_path)
    if spec is None or spec.loader is None:
        raise RuntimeError(f"Unable to load module from {module_path}")
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


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

    assert payload == {"ok": True}
    assert headers == {"x-test": "1"}
    assert observed["host"] == "api.github.com"
    assert observed["request_target"] == "/repos/owner/repo"


def test_check_required_checks_api_get_uses_secure_helper(monkeypatch: pytest.MonkeyPatch) -> None:
    module = _load_module("check_required_checks_under_test", "scripts/quality/check_required_checks.py")

    observed: dict[str, object] = {}

    def fake_request_https_json(url: str, **kwargs):
        observed["url"] = url
        observed.update(kwargs)
        return {"ok": True}, {"x-test": "1"}

    monkeypatch.setattr(module, "request_https_json", fake_request_https_json, raising=False)

    payload = module._api_get("owner/repo", "commits/abc/status", "token")

    assert payload == {"ok": True}
    assert observed["url"] == "https://api.github.com/repos/owner/repo/commits/abc/status"
    assert observed["allowed_hosts"] == {"api.github.com"}
    assert observed["method"] == "GET"
