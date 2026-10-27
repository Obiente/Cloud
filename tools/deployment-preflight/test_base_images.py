import contextlib
import io
import json
import os
from pathlib import Path
import subprocess
import tempfile
import unittest
from unittest.mock import patch

import base_images


class BaseImageTests(unittest.TestCase):
    def test_reads_all_stages_without_duplicate_images_or_stage_aliases(self):
        dockerfile = """
ARG BASE_REGISTRY=docker.io
FROM --platform=$BUILDPLATFORM ${BASE_REGISTRY}/library/golang:1.27.3-alpine AS build
FROM build AS test
FROM \\
  $BASE_REGISTRY/library/alpine:latest AS runtime
FROM ${BASE_REGISTRY}/library/alpine:latest AS final
"""
        self.assertEqual(
            base_images.base_images(dockerfile),
            ["library/golang:1.27.3-alpine", "library/alpine:latest"],
        )

    def test_checks_actual_repository_dockerfiles(self):
        root = Path(__file__).resolve().parents[2]
        files = list((root / "apps").glob("*/Dockerfile"))
        self.assertGreaterEqual(len(files), 17)
        for file in files:
            with self.subTest(dockerfile=file.parent.name):
                self.assertTrue(base_images.base_images(file.read_text(encoding="utf-8")))

    def test_unsupported_dynamic_reference_fails(self):
        with self.assertRaises(ValueError):
            base_images.base_images("FROM ${BASE_REGISTRY}/library/golang:${GO_VERSION}")

    def test_missing_base_reference_fails(self):
        with self.assertRaises(ValueError):
            base_images.base_images("FROM scratch")

    def test_uses_mirror_when_every_image_exists(self):
        calls = []
        result = base_images.choose_registry(
            ["library/golang:test", "library/alpine:test"],
            "linux/amd64",
            "mirror.example.test/dockerhub",
            probe=lambda image, platform: calls.append((image, platform)) or True,
        )
        self.assertEqual(result, "mirror.example.test/dockerhub")
        self.assertEqual(len(calls), 2)

    def test_missing_new_go_tag_uses_upstream_for_the_complete_build(self):
        calls = []

        def probe(image, platform):
            calls.append((image, platform))
            return image != "mirror.example.test/dockerhub/library/golang:1.27.3-alpine"

        refs = ["library/golang:1.27.3-alpine", "library/alpine:latest"]
        with contextlib.redirect_stderr(io.StringIO()):
            result = base_images.choose_registry(
                refs, "linux/arm64", "mirror.example.test/dockerhub", probe=probe
            )
        self.assertEqual(result, "docker.io")
        self.assertEqual(calls[-2:], [("docker.io/" + ref, "linux/arm64") for ref in refs])

    def test_no_registry_with_complete_inventory_fails(self):
        with contextlib.redirect_stderr(io.StringIO()), self.assertRaises(ValueError):
            base_images.choose_registry(
                ["library/golang:test"], "linux/amd64", "mirror.example.test",
                probe=lambda *_: False,
            )

    def test_public_validation_does_not_probe_the_same_registry_twice(self):
        calls = []
        with contextlib.redirect_stderr(io.StringIO()), self.assertRaises(ValueError):
            base_images.choose_registry(
                ["library/golang:test"], "linux/amd64", "docker.io",
                probe=lambda *args: calls.append(args) or False,
            )
        self.assertEqual(len(calls), 1)

    def test_checks_architecture_and_ignores_attestations(self):
        payload = {"manifests": [
            {"platform": {"os": "linux", "architecture": "amd64"}},
            {"platform": {"os": "unknown", "architecture": "unknown"}},
        ]}
        result = subprocess.CompletedProcess([], 0, json.dumps(payload))
        with patch.object(base_images.subprocess, "run", return_value=result) as run:
            self.assertTrue(base_images.supports_platform("mirror.example.test/image:test", "linux/amd64"))
            self.assertFalse(base_images.supports_platform("mirror.example.test/image:test", "linux/arm64"))
        self.assertEqual(run.call_args.args[0], [
            "docker", "buildx", "imagetools", "inspect", "--raw",
            "mirror.example.test/image:test",
        ])

    def test_invalid_or_unavailable_manifest_is_not_accepted(self):
        results = [
            subprocess.CompletedProcess([], 1, ""),
            subprocess.CompletedProcess([], 0, "not json"),
            subprocess.CompletedProcess([], 0, '{"schemaVersion":2}'),
        ]
        for result in results:
            with self.subTest(result=result), patch.object(
                base_images.subprocess, "run", return_value=result
            ):
                self.assertFalse(base_images.supports_platform(
                    "mirror.example.test/image:test", "linux/arm64"
                ))
        with patch.object(
            base_images.subprocess, "run", side_effect=subprocess.TimeoutExpired("docker", 45)
        ):
            self.assertFalse(base_images.supports_platform(
                "mirror.example.test/image:test", "linux/arm64"
            ))

    def test_failed_preflight_never_writes_build_output(self):
        with tempfile.TemporaryDirectory() as directory:
            dockerfile = Path(directory) / "Dockerfile"
            dockerfile.write_text("FROM ${BASE_REGISTRY}/library/golang:test", encoding="utf-8")
            output = Path(directory) / "output"
            with (
                patch.dict(os.environ, {"GITHUB_OUTPUT": str(output)}),
                patch.object(base_images, "choose_registry", side_effect=ValueError("unavailable")),
                contextlib.redirect_stderr(io.StringIO()),
            ):
                self.assertEqual(base_images.main([
                    "--dockerfile", str(dockerfile), "--platform", "linux/amd64",
                    "--mirror", "mirror.example.test",
                ]), 1)
            self.assertFalse(output.exists())

    def test_success_emits_selected_registry(self):
        with tempfile.TemporaryDirectory() as directory:
            dockerfile = Path(directory) / "Dockerfile"
            dockerfile.write_text("FROM ${BASE_REGISTRY}/library/golang:test", encoding="utf-8")
            output = Path(directory) / "output"
            with (
                patch.dict(os.environ, {"GITHUB_OUTPUT": str(output)}),
                patch.object(base_images, "choose_registry", return_value="docker.io"),
                contextlib.redirect_stdout(io.StringIO()),
            ):
                self.assertEqual(base_images.main([
                    "--dockerfile", str(dockerfile), "--platform", "linux/amd64",
                    "--mirror", "mirror.example.test",
                ]), 0)
            self.assertEqual(output.read_text(encoding="utf-8"), "registry=docker.io\n")


if __name__ == "__main__":
    unittest.main()
