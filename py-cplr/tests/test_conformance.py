"""Conformance suite: runs the language-neutral vectors in ../spec/cases.yaml."""

import json
import os

import pytest
import yaml as pyyaml

from sdui_template_compiler import ScreenRegistry
from sdui_template_compiler.json_compat import stringify

SPEC_DIR = os.path.join(os.path.dirname(__file__), "..", "..", "spec")
FIXTURES_DIR = os.path.join(SPEC_DIR, "fixtures")

with open(os.path.join(SPEC_DIR, "cases.yaml"), "r", encoding="utf-8") as handle:
    CASES = pyyaml.safe_load(handle)


def _resolve_template_version(registry: ScreenRegistry, id: str, template_version: str):
    module = registry.module_of(id)
    assert module is not None, f"Unknown screen: {id}"
    threshold = next(
        (
            version
            for version, assets in module.versions.items()
            if assets.template == template_version
        ),
        None,
    )
    assert threshold is not None, f"No threshold maps to template {template_version}"
    composed = registry.resolve(id, threshold)
    assert composed is not None, f"Failed to resolve {id}"
    return composed


class TestGolden:
    @pytest.mark.parametrize("fixture", CASES["golden"])
    def test_composes_byte_identically(self, fixture: str) -> None:
        input_dir = os.path.join(FIXTURES_DIR, fixture, "input")
        expected_dir = os.path.join(FIXTURES_DIR, fixture, "expected")
        registry = ScreenRegistry.build(input_dir)
        with open(os.path.join(expected_dir, "manifest.json"), "r", encoding="utf-8") as f:
            manifest = json.load(f)

        for id in manifest:
            screen_dir = os.path.join(expected_dir, "screens", id)
            for file in os.listdir(screen_dir):
                template_version = file.removesuffix(".json")
                composed = _resolve_template_version(registry, id, template_version)
                with open(os.path.join(screen_dir, file), "r", encoding="utf-8") as f:
                    expected = f.read()
                assert stringify(composed.template) == expected
                assert composed.etag == manifest[id]["etags"][template_version]

        # The CLI manifest must be byte-identical too: discovery order,
        # pretty serialization, and etags all locked across ports.
        built: dict = {}
        for id in registry.ids():
            module = registry.module_of(id)
            assert module is not None, f"Unknown screen: {id}"
            etags: dict[str, str] = {}
            for assets in module.versions.values():
                if assets.template in etags:
                    continue
                etags[assets.template] = _resolve_template_version(
                    registry, id, assets.template
                ).etag
            built[id] = {
                "versions": {v: {"template": a.template} for v, a in module.versions.items()},
                "params": list(module.params),
                "etags": etags,
            }
        with open(os.path.join(expected_dir, "manifest.json"), "r", encoding="utf-8") as f:
            assert stringify(built, indent=2) == f.read()


class TestVersioning:
    @pytest.mark.parametrize(
        "query", CASES["versioning"]["queries"], ids=lambda q: str(q["app_version"])
    )
    def test_selects_template(self, query: dict) -> None:
        spec = CASES["versioning"]
        input_dir = os.path.join(FIXTURES_DIR, spec["fixture"], "input")
        registry = ScreenRegistry.build(input_dir)
        composed = registry.resolve(spec["screen"], query["app_version"])
        via_threshold = _resolve_template_version(registry, spec["screen"], query["template"])
        assert composed is not None
        assert composed.etag == via_threshold.etag

    @pytest.mark.parametrize("invalid", CASES["versioning"]["invalid_app_versions"])
    def test_rejects_malformed_app_version(self, invalid: str) -> None:
        spec = CASES["versioning"]
        input_dir = os.path.join(FIXTURES_DIR, spec["fixture"], "input")
        registry = ScreenRegistry.build(input_dir)
        with pytest.raises(Exception, match="Invalid semantic version"):
            registry.resolve(spec["screen"], invalid)


class TestErrors:
    @pytest.mark.parametrize(
        ("fixture", "message"), sorted(CASES["errors"].items()), ids=lambda v: str(v)
    )
    def test_rejects_fixture(self, fixture: str, message: str) -> None:
        input_dir = os.path.join(FIXTURES_DIR, "errors", fixture, "input")
        with pytest.raises(Exception) as excinfo:
            ScreenRegistry.build(input_dir)
        assert message in str(excinfo.value)
