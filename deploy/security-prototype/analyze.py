"""Static-only Pickle analysis in a resource-limited child process.

Copyright The MatrixHub Authors. Licensed under Apache-2.0.
Fickling is an external LGPL-3.0 dependency; no Fickling code is vendored here.
"""
import ast
import io
import json
import pickletools
import resource
import sys

resource.setrlimit(resource.RLIMIT_AS, (256 * 1024 * 1024,) * 2)
resource.setrlimit(resource.RLIMIT_CPU, (8, 8))
resource.setrlimit(resource.RLIMIT_NOFILE, (32, 32))
resource.setrlimit(resource.RLIMIT_FSIZE, (0, 0))

from fickling.analysis import check_safety
from fickling.fickle import Pickled


def analyze(data):
    count = 0
    stop = -1
    # These opcodes construct only literal containers/scalars and memo entries;
    # GLOBAL, REDUCE, BUILD, EXT and persistent references are deliberately absent.
    literal_ops = {"PROTO", "FRAME", "STOP", "MARK", "POP", "POP_MARK", "DUP",
                   "NONE", "NEWTRUE", "NEWFALSE", "INT", "BININT", "BININT1", "BININT2", "LONG", "LONG1", "LONG4", "FLOAT", "BINFLOAT",
                   "STRING", "BINSTRING", "SHORT_BINSTRING", "UNICODE", "BINUNICODE", "SHORT_BINUNICODE", "BINUNICODE8", "BINBYTES", "SHORT_BINBYTES", "BINBYTES8", "BYTEARRAY8",
                   "EMPTY_LIST", "LIST", "APPEND", "APPENDS", "EMPTY_TUPLE", "TUPLE", "TUPLE1", "TUPLE2", "TUPLE3", "EMPTY_DICT", "DICT", "SETITEM", "SETITEMS", "EMPTY_SET", "ADDITEMS", "FROZENSET",
                   "PUT", "BINPUT", "LONG_BINPUT", "GET", "BINGET", "LONG_BINGET", "MEMOIZE"}
    data_only = True
    # Preflight validates framing without reconstructing objects or importing globals.
    for opcode, argument, offset in pickletools.genops(data):
        count += 1
        data_only = data_only and opcode.name in literal_ops
        if count > 100_000:
            raise ValueError("opcode budget")
        if opcode.name == "PROTO" and argument > 5:
            raise ValueError("unsupported protocol")
        if opcode.name == "STOP":
            stop = offset
    if stop != len(data) - 1:
        raise ValueError("incomplete or trailing pickle stream")
    pickled = Pickled.load(io.BytesIO(data))
    result = check_safety(pickled)
    severity = result.severity.severity
    # Protocol 0 container reconstruction can trip Fickling's unused-variable
    # heuristic. Suppress only that isolated heuristic for a no-call opcode stream.
    if data_only and result.results and all(r.analysis_name == "UnusedVariables" and r.severity.severity < 3 for r in result.results):
        return {"status": "passed", "rule": "DATA_ONLY_NO_CALLS", "severity": "info"}
    # A narrow ML construction profile distinguishes expected reconstruction
    # from arbitrary-code primitives. It remains a review warning, never a
    # clean verdict: runtime dependencies and persistent storage are not loaded.
    review_imports = {("torch._utils", "_rebuild_tensor_v2"), ("torch", "FloatStorage"),
                      ("torch", "LongStorage"), ("collections", "OrderedDict")}
    imports = pickled.properties.imports
    calls = pickled.properties.calls
    known_imports = bool(imports) and all(isinstance(node, ast.ImportFrom) and node.level == 0 and
        all((node.module, alias.name) in review_imports and alias.asname is None for alias in node.names) for node in imports)
    known_calls = all(isinstance(node.func, ast.Name) and node.func.id in {"_rebuild_tensor_v2", "OrderedDict"} for node in calls)
    known_heuristics = all(r.analysis_name in {"NonStandardImports", "OvertlyBadEval", "UnusedVariables"} for r in result.results)
    if severity >= 3 and known_imports and known_calls and known_heuristics:
        return {"status": "warning", "rule": "ML_CONSTRUCTION_REVIEW", "severity": "medium"}
    return {
        "status": "blocked" if severity >= 3 else "warning" if severity else "passed",
        # Do not forward reconstructed code, arguments or analyzer exception text.
        "rule": result.severity.name,
        "severity": "high" if severity >= 3 else "medium" if severity else "info",
    }


if __name__ == "__main__":
    try:
        content = sys.stdin.buffer.read(8 * 1024 * 1024 + 1)
        if len(content) > 8 * 1024 * 1024:
            raise ValueError("size budget")
        print(json.dumps(analyze(content)))
    except Exception:
        print(json.dumps({"status": "failed", "rule": "STATIC_ANALYSIS_INCOMPLETE", "severity": "unknown"}))
        sys.exit(2)
