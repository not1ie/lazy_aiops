#!/usr/bin/env python3
"""
Verify frontend API usages against backend registered routes.

Usage:
  python3 scripts/verify_frontend_backend_api.py
"""

from __future__ import annotations

import argparse
import pathlib
import re
import sys
from dataclasses import dataclass
from typing import Dict, Iterable, List, Optional, Set, Tuple


HTTP_METHODS = {"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS", "HEAD"}


@dataclass(frozen=True)
class RouteRef:
    method: str
    path: str
    source: str


@dataclass(frozen=True)
class FrontRef:
    method: str
    path: str
    source: str
    lineno: int


@dataclass(frozen=True)
class ClickRef:
    source: str
    lineno: int
    handler: str
    expr: str


def strip_quotes(value: str) -> str:
    value = value.strip()
    if len(value) >= 2 and value[0] in ("'", '"', "`") and value[-1] == value[0]:
        return value[1:-1]
    return value


def normalize_path(raw: str) -> str:
    path = (raw or "").strip()
    if not path:
        return ""
    if "?" in path:
        path = path.split("?", 1)[0]
    if "://" in path:
        idx = path.find("/api/v1/")
        if idx >= 0:
            path = path[idx:]
    if "/api/v1/" not in path:
        return ""
    path = path[path.find("/api/v1/") :]
    path = re.sub(r"\$\{[^}]+\}", ":param", path)
    path = re.sub(r"/{2,}", "/", path)
    if not path.startswith("/"):
        path = "/" + path
    parts: List[str] = []
    for seg in path.split("/"):
        if not seg:
            continue
        if seg.startswith(":"):
            parts.append(":param")
        else:
            parts.append(seg)
    return "/" + "/".join(parts)


def join_path(prefix: str, suffix: str) -> str:
    base = prefix.rstrip("/")
    extra = suffix if suffix.startswith("/") else f"/{suffix}"
    return normalize_path(f"{base}{extra}")


def same_pattern(a: str, b: str) -> bool:
    sa = [s for s in a.strip("/").split("/") if s]
    sb = [s for s in b.strip("/").split("/") if s]
    if len(sa) != len(sb):
        return False
    for x, y in zip(sa, sb):
        if x == ":param" or y == ":param":
            continue
        if x != y:
            return False
    return True


def extract_block_body(content: str, start_idx: int) -> str:
    brace = content.find("{", start_idx)
    if brace < 0:
        return ""
    depth = 0
    for i in range(brace, len(content)):
        ch = content[i]
        if ch == "{":
            depth += 1
        elif ch == "}":
            depth -= 1
            if depth == 0:
                return content[brace + 1 : i]
    return ""


def extract_function_body(content: str, signature: str) -> str:
    start = content.find(signature)
    if start < 0:
        return ""
    return extract_block_body(content, start)


def parse_register_routes(plugin_file: pathlib.Path, plugin_name: str) -> List[RouteRef]:
    text = plugin_file.read_text(encoding="utf-8", errors="ignore")
    sig = re.search(r"func\s*\([^)]*\)\s*RegisterRoutes\(\s*(\w+)\s+\*gin\.RouterGroup\s*\)", text)
    if not sig:
        return []
    root_group_var = sig.group(1)
    body = extract_block_body(text, sig.end())
    if not body:
        return []

    prefixes: Dict[str, str] = {root_group_var: ""}
    routes: List[RouteRef] = []
    group_re = re.compile(r"(\w+)\s*:=\s*(\w+)\.Group\(\s*\"([^\"]*)\"\s*\)")
    route_re = re.compile(r"(\w+)\.(GET|POST|PUT|DELETE|PATCH|OPTIONS|HEAD)\(\s*\"([^\"]*)\"")

    for raw_line in body.splitlines():
        line = raw_line.split("//", 1)[0].strip()
        if not line:
            continue
        gm = group_re.search(line)
        if gm:
            child, parent, suffix = gm.groups()
            parent_prefix = prefixes.get(parent, "")
            merged = join_path(f"/api/v1/{plugin_name}{parent_prefix}", suffix)
            # prefixes 保存插件内相对前缀
            rel = merged.replace(f"/api/v1/{plugin_name}", "", 1)
            prefixes[child] = rel if rel else ""
        rm = route_re.search(line)
        if rm:
            group_var, method, suffix = rm.groups()
            rel_prefix = prefixes.get(group_var, "")
            full = join_path(f"/api/v1/{plugin_name}{rel_prefix}", suffix)
            if full:
                routes.append(RouteRef(method=method, path=full, source=str(plugin_file)))
    return routes


def parse_server_routes(server_file: pathlib.Path) -> List[RouteRef]:
    text = server_file.read_text(encoding="utf-8", errors="ignore")
    routes: List[RouteRef] = []

    for fn in ("setupPublicRoutes(g *gin.RouterGroup)", "setupAuthRoutes(g *gin.RouterGroup)"):
        body = extract_function_body(text, f"func (s *Server) {fn}")
        if not body:
            continue
        route_re = re.compile(r"\bg\.(GET|POST|PUT|DELETE|PATCH|OPTIONS|HEAD)\(\s*\"([^\"]*)\"")
        for m in route_re.finditer(body):
            method, suffix = m.groups()
            full = join_path("/api/v1", suffix)
            if full:
                routes.append(RouteRef(method=method, path=full, source=str(server_file)))
    return routes


def parse_plugin_names(plugin_root: pathlib.Path) -> Dict[pathlib.Path, str]:
    result: Dict[pathlib.Path, str] = {}
    for plugin_file in sorted(plugin_root.glob("*/plugin.go")):
        text = plugin_file.read_text(encoding="utf-8", errors="ignore")
        m = re.search(r'func\s*\(p\s*\*\w+\)\s*Name\(\)\s*string\s*\{\s*return\s*"([^"]+)"\s*\}', text)
        if m:
            result[plugin_file] = m.group(1)
    return result


def parse_frontend_refs(front_root: pathlib.Path) -> List[FrontRef]:
    refs: List[FrontRef] = []
    files = list(front_root.rglob("*.vue")) + list(front_root.rglob("*.js")) + list(front_root.rglob("*.ts"))

    axios_call_re = re.compile(
        r"axios\.(get|post|put|delete|patch|options|head)\(\s*(`[^`]*`|'[^']*'|\"[^\"]*\")",
        re.IGNORECASE,
    )
    axios_obj_re = re.compile(r"axios\(\s*\{([\s\S]{0,600}?)\}\s*\)", re.IGNORECASE)
    ws_re = re.compile(r"new\s+WebSocket\(\s*(`[^`]*`|'[^']*'|\"[^\"]*\")")
    string_api_re = re.compile(r"(`[^`]*\/api\/v1\/[^`]*`|'[^']*\/api\/v1\/[^']*'|\"[^\"]*\/api\/v1\/[^\"]*\")")

    for file in files:
        text = file.read_text(encoding="utf-8", errors="ignore")
        lines = text.splitlines()
        indexed = "\n".join(lines)

        for m in axios_call_re.finditer(indexed):
            method = m.group(1).upper()
            raw = strip_quotes(m.group(2))
            path = normalize_path(raw)
            if not path:
                continue
            lineno = indexed.count("\n", 0, m.start()) + 1
            refs.append(FrontRef(method=method, path=path, source=str(file), lineno=lineno))

        for m in axios_obj_re.finditer(indexed):
            block = m.group(1)
            method_match = re.search(r"method\s*:\s*['\"](get|post|put|delete|patch|options|head)['\"]", block, re.IGNORECASE)
            method = method_match.group(1).upper() if method_match else "GET"
            for s in string_api_re.finditer(block):
                raw = strip_quotes(s.group(1))
                path = normalize_path(raw)
                if not path:
                    continue
                lineno = indexed.count("\n", 0, m.start()) + 1
                refs.append(FrontRef(method=method, path=path, source=str(file), lineno=lineno))

        for m in ws_re.finditer(indexed):
            raw = strip_quotes(m.group(1))
            path = normalize_path(raw)
            if not path:
                continue
            lineno = indexed.count("\n", 0, m.start()) + 1
            refs.append(FrontRef(method="GET", path=path, source=str(file), lineno=lineno))

    # 去重
    uniq: Dict[Tuple[str, str, str, int], FrontRef] = {}
    for r in refs:
        uniq[(r.method, r.path, r.source, r.lineno)] = r
    return list(uniq.values())


def parse_click_handler_refs(front_root: pathlib.Path) -> List[ClickRef]:
    issues: List[ClickRef] = []
    for file in sorted(front_root.rglob("*.vue")):
        text = file.read_text(encoding="utf-8", errors="ignore")
        template = text.split("<script", 1)[0]

        # <script setup>
        script_setup = ""
        setup_match = re.search(r"<script\s+setup[^>]*>([\s\S]*?)</script>", text)
        if setup_match:
            script_setup = setup_match.group(1)

        setup_defs: Set[str] = set()
        if script_setup:
            setup_defs.update(
                re.findall(
                    r"\bconst\s+([A-Za-z_][A-Za-z0-9_]*)\s*=\s*(?:async\s*)?(?:\([^)]*\)\s*=>|function\b)",
                    script_setup,
                )
            )
            setup_defs.update(re.findall(r"\bfunction\s+([A-Za-z_][A-Za-z0-9_]*)\s*\(", script_setup))

        # options api methods
        options_defs: Set[str] = set()
        script_match = re.search(r"<script(?!\s+setup)[^>]*>([\s\S]*?)</script>", text)
        if script_match:
            script = script_match.group(1)
            methods_match = re.search(r"methods\s*:\s*\{([\s\S]*?)\}\s*(?:,|\n|\r)", script)
            if methods_match:
                methods_body = methods_match.group(1)
                options_defs.update(re.findall(r"\b([A-Za-z_][A-Za-z0-9_]*)\s*\(", methods_body))

        defs = setup_defs | options_defs

        click_re = re.compile(r'@click(?:\.[a-z]+)*\s*=\s*"([^"]+)"')
        for m in click_re.finditer(template):
            expr = m.group(1).strip()
            if not expr:
                continue
            # skip inline assignment / mutation expressions
            if "=" in expr and "==" not in expr and "!=" not in expr:
                continue
            ident = re.match(r"([A-Za-z_][A-Za-z0-9_]*)\s*(?:\(|$)", expr)
            if not ident:
                continue
            handler = ident.group(1)
            if handler in {"$emit", "$router", "$refs"}:
                continue
            if handler not in defs:
                lineno = text.count("\n", 0, m.start()) + 1
                issues.append(
                    ClickRef(
                        source=str(file),
                        lineno=lineno,
                        handler=handler,
                        expr=expr,
                    )
                )
    return issues


def find_match(front: FrontRef, backend: Iterable[RouteRef]) -> bool:
    for route in backend:
        if route.method != front.method:
            continue
        if same_pattern(route.path, front.path):
            return True
    return False


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--root", default=".", help="project root")
    args = parser.parse_args()

    root = pathlib.Path(args.root).resolve()
    plugin_root = root / "plugins"
    server_file = root / "internal" / "api" / "server.go"
    front_root = root / "frontend" / "src"

    if not plugin_root.exists() or not server_file.exists() or not front_root.exists():
        print("project layout not found", file=sys.stderr)
        return 2

    backend_routes: List[RouteRef] = []
    backend_routes.extend(parse_server_routes(server_file))
    plugin_names = parse_plugin_names(plugin_root)
    for plugin_file, plugin_name in plugin_names.items():
        backend_routes.extend(parse_register_routes(plugin_file, plugin_name))

    # 去重
    backend_routes = list({(r.method, r.path, r.source): r for r in backend_routes}.values())

    front_refs = parse_frontend_refs(front_root)
    click_issues = parse_click_handler_refs(front_root)
    unmatched: List[FrontRef] = []
    for ref in front_refs:
        if not find_match(ref, backend_routes):
            unmatched.append(ref)

    print(f"Backend routes: {len(backend_routes)}")
    print(f"Frontend api refs: {len(front_refs)}")
    print(f"Unmatched refs: {len(unmatched)}")
    print(f"Missing click handlers: {len(click_issues)}")

    if unmatched:
        print("\nPossible broken buttons / API calls:")
        for u in sorted(unmatched, key=lambda x: (x.source, x.lineno, x.method, x.path)):
            print(f"- {u.source}:{u.lineno}  {u.method} {u.path}")
    if click_issues:
        print("\nPossible broken click handlers:")
        for u in sorted(click_issues, key=lambda x: (x.source, x.lineno, x.handler)):
            print(f"- {u.source}:{u.lineno}  {u.handler} <- {u.expr}")

    if unmatched or click_issues:
        return 1

    print("\nNo static route mismatches or missing click handlers found.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
