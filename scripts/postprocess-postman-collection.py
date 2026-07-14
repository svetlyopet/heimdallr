#!/usr/bin/env python3
"""Post-process an OpenAPI-generated Postman collection for Heimdallr."""

from __future__ import annotations

import json
import sys
from collections import defaultdict
from pathlib import Path

import yaml

TAG_ORDER = [
    "Health",
    "Application",
    "Release",
    "Report",
    "Provider",
    "Automation",
    "Job",
    "Server",
    "Agent",
    "Analytics",
    "Auth",
    "Token",
]

PATH_VAR_DEFAULTS = {
    "provider_id": "00000000-0000-0000-0000-000000000001",
    "automation_id": "00000000-0000-0000-0000-000000000002",
    "job_id": "example-job-id",
    "application_id": "00000000-0000-0000-0000-000000000003",
    "release_id": "00000000-0000-0000-0000-000000000004",
    "report_id": "00000000-0000-0000-0000-000000000005",
    "server_id": "00000000-0000-0000-0000-000000000006",
    "agent_id": "00000000-0000-0000-0000-000000000007",
    "user_id": "00000000-0000-0000-0000-000000000008",
    "token_id": "00000000-0000-0000-0000-000000000009",
}

LOGIN_TEST_SCRIPT = """pm.test("Login successful", function () {
    pm.response.to.have.status(200);
});

if (pm.response.code === 200) {
    const json = pm.response.json();
    if (json.data && json.data.token) {
        pm.collectionVariables.set("bearer_token", json.data.token);
    }
}
"""

COLLECTION_VARIABLES = [
    {"key": "baseUrl", "value": "http://localhost:8080/api", "type": "string"},
    {"key": "bearer_token", "value": "", "type": "string"},
    {"key": "provider_id", "value": PATH_VAR_DEFAULTS["provider_id"], "type": "string"},
    {"key": "automation_id", "value": PATH_VAR_DEFAULTS["automation_id"], "type": "string"},
    {"key": "job_id", "value": PATH_VAR_DEFAULTS["job_id"], "type": "string"},
    {"key": "application_id", "value": PATH_VAR_DEFAULTS["application_id"], "type": "string"},
    {"key": "release_id", "value": PATH_VAR_DEFAULTS["release_id"], "type": "string"},
    {"key": "report_id", "value": PATH_VAR_DEFAULTS["report_id"], "type": "string"},
    {"key": "server_id", "value": PATH_VAR_DEFAULTS["server_id"], "type": "string"},
    {"key": "agent_id", "value": PATH_VAR_DEFAULTS["agent_id"], "type": "string"},
    {"key": "user_id", "value": PATH_VAR_DEFAULTS["user_id"], "type": "string"},
    {"key": "token_id", "value": PATH_VAR_DEFAULTS["token_id"], "type": "string"},
]


def load_tag_mapping(openapi_path: Path) -> dict[str, str]:
    with openapi_path.open(encoding="utf-8") as handle:
        spec = yaml.safe_load(handle)

    mapping: dict[str, str] = {}
    for methods in spec.get("paths", {}).values():
        for method, operation in methods.items():
            if method not in {"get", "post", "put", "patch", "delete"}:
                continue
            tag = operation.get("tags", ["Other"])[0]
            summary = operation.get("summary", "")
            operation_id = operation.get("operationId", "")
            if summary:
                mapping[summary] = tag
            if operation_id:
                mapping[operation_id] = tag
    return mapping


def iter_requests(items: list[dict]) -> list[dict]:
    requests: list[dict] = []
    for item in items:
        if "request" in item:
            requests.append(item)
            continue
        if "item" in item:
            requests.extend(iter_requests(item["item"]))
    return requests


def normalize_request(item: dict, tag_mapping: dict[str, str]) -> dict:
    request = item["request"]
    name = item.get("name", request.get("name", "Request"))
    tag = tag_mapping.get(name, "Other")

    url = request.get("url", {})
    for variable in url.get("variable", []):
        key = variable.get("key", "")
        if key in PATH_VAR_DEFAULTS:
            variable["value"] = f"{{{{{key}}}}}"

    headers = []
    for header in request.get("header", []):
        if header.get("key") == "Authorization":
            continue
        headers.append(header)
    request["header"] = headers

    method = request.get("method", "GET")
    path = url.get("path", [])
    is_health = path == ["health"]
    is_login = path == ["v1", "auth", "login"] and method == "POST"

    if is_health or is_login:
        request["auth"] = None
    else:
        request["auth"] = {
            "type": "bearer",
            "bearer": [
                {"key": "token", "value": "{{bearer_token}}", "type": "string"},
            ],
        }

    if method in {"POST", "PUT", "PATCH"}:
        has_content_type = any(h.get("key") == "Content-Type" for h in headers)
        if not has_content_type:
            request["header"].append(
                {"key": "Content-Type", "value": "application/json"},
            )

    cleaned = {
        "name": name,
        "request": request,
        "response": [],
    }

    if is_login:
        cleaned["event"] = [
            {
                "listen": "test",
                "script": {
                    "type": "text/javascript",
                    "exec": LOGIN_TEST_SCRIPT.splitlines(),
                },
            },
        ]

    cleaned["_tag"] = tag
    return cleaned


def build_folders(requests: list[dict]) -> list[dict]:
    grouped: dict[str, list[dict]] = defaultdict(list)
    for item in requests:
        tag = item.pop("_tag", "Other")
        grouped[tag].append(item)

    folders: list[dict] = []
    for tag in TAG_ORDER:
        if tag not in grouped:
            continue
        folders.append({"name": tag, "item": grouped.pop(tag)})

    for tag in sorted(grouped):
        folders.append({"name": tag, "item": grouped[tag]})

    return folders


def postprocess(generated_path: Path, openapi_path: Path, output_path: Path) -> None:
    tag_mapping = load_tag_mapping(openapi_path)

    with generated_path.open(encoding="utf-8") as handle:
        generated = json.load(handle)

    raw_requests = iter_requests(generated.get("item", []))
    requests = [normalize_request(item, tag_mapping) for item in raw_requests]

    info = generated.get("info", {})
    description = info.get("description", {})
    if isinstance(description, dict):
        description = description.get("content", "")

    collection = {
        "info": {
            "name": info.get("name", "Heimdallr API"),
            "description": description,
            "schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json",
            "version": "1.0.0",
        },
        "auth": {
            "type": "bearer",
            "bearer": [
                {"key": "token", "value": "{{bearer_token}}", "type": "string"},
            ],
        },
        "variable": COLLECTION_VARIABLES,
        "item": build_folders(requests),
    }

    output_path.parent.mkdir(parents=True, exist_ok=True)
    with output_path.open("w", encoding="utf-8") as handle:
        json.dump(collection, handle, indent=2)
        handle.write("\n")

    print(f"Wrote {len(requests)} requests to {output_path}")


def main() -> None:
    repo_root = Path(__file__).resolve().parents[1]
    generated = Path(sys.argv[1]) if len(sys.argv) > 1 else Path("/tmp/heimdallr_postman_generated.json")
    output = Path(sys.argv[2]) if len(sys.argv) > 2 else repo_root / "api" / "postman_collection.json"
    openapi = repo_root / "api" / "docs" / "openapi.yaml"
    postprocess(generated, openapi, output)


if __name__ == "__main__":
    main()
