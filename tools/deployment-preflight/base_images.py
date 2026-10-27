#!/usr/bin/env python3
"""Choose a reachable registry for the Dockerfile's declared base-image tags."""

import argparse
import json
import os
from pathlib import Path
import re
import subprocess
import sys


def base_images(dockerfile):
    content = re.sub(r"\\\r?\n", " ", dockerfile)
    refs = []
    for line in content.splitlines():
        match = re.match(
            r"^\s*FROM\s+(?:--platform=\S+\s+)?(?:\$\{BASE_REGISTRY\}|\$BASE_REGISTRY)/(\S+)",
            line,
            re.IGNORECASE,
        )
        if match:
            ref = match.group(1)
            if "$" in ref or not re.fullmatch(r"[a-z0-9][A-Za-z0-9._/:@-]*", ref):
                raise ValueError(f"Unsupported base-image reference: {ref}")
            if ref not in refs:
                refs.append(ref)
    if not refs:
        raise ValueError("Dockerfile has no BASE_REGISTRY image references")
    return refs


def supports_platform(image, platform):
    try:
        result = subprocess.run(
            ["docker", "buildx", "imagetools", "inspect", "--raw", image],
            capture_output=True,
            text=True,
            timeout=45,
            check=False,
        )
        if result.returncode:
            return False
        manifest = json.loads(result.stdout)
        os_name, architecture, *variant = platform.split("/")
        return any(
            entry.get("platform", {}).get("os") == os_name
            and entry.get("platform", {}).get("architecture") == architecture
            and (not variant or entry.get("platform", {}).get("variant") == variant[0])
            for entry in manifest.get("manifests", [])
        )
    except (subprocess.TimeoutExpired, OSError, ValueError, AttributeError, TypeError):
        return False


def choose_registry(refs, platform, mirror, upstream="docker.io", probe=supports_platform):
    for registry in dict.fromkeys((mirror.rstrip("/"), upstream.rstrip("/"))):
        missing = [ref for ref in refs if not probe(f"{registry}/{ref}", platform)]
        if not missing:
            return registry
        print(
            f"Base images unavailable for {platform} in {registry}: {', '.join(missing)}",
            file=sys.stderr,
        )
    raise ValueError(f"Required base images are unavailable for {platform}; build stopped")


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--dockerfile", required=True, type=Path)
    parser.add_argument("--platform", required=True, choices=("linux/amd64", "linux/arm64"))
    parser.add_argument("--mirror", required=True)
    args = parser.parse_args(argv)
    try:
        refs = base_images(args.dockerfile.read_text(encoding="utf-8"))
        registry = choose_registry(refs, args.platform, args.mirror)
    except (OSError, ValueError) as error:
        print(error, file=sys.stderr)
        return 1
    print(f"Using {registry} base images for {args.platform}")
    if output := os.environ.get("GITHUB_OUTPUT"):
        with open(output, "a", encoding="utf-8") as stream:
            stream.write(f"registry={registry}\n")
    return 0


if __name__ == "__main__":
    sys.exit(main())
