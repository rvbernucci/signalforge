from __future__ import annotations

import importlib.util
from pathlib import Path
import unittest


MODULE_PATH = Path(__file__).resolve().parents[1] / "ir_lineage.py"
SPEC = importlib.util.spec_from_file_location("ir_lineage", MODULE_PATH)
assert SPEC and SPEC.loader
MODULE = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(MODULE)


class IRLineageTests(unittest.TestCase):
    def test_missing_current_is_not_called_deleted(self) -> None:
        previous = [{"canonical_uri": "https://example.com/a", "document_id": "a", "content_sha256": "1", "retrieved_at": "old"}]
        records = MODULE.compare(previous, [])
        self.assertEqual("not_observed_in_current_run", records[0]["disposition"])

    def test_same_uri_with_new_hash_is_content_change(self) -> None:
        previous = [{"canonical_uri": "https://example.com/a", "document_id": "a", "content_sha256": "1", "retrieved_at": "old"}]
        current = [{"canonical_uri": "https://example.com/a", "document_id": "a", "content_sha256": "2", "retrieved_at": "new"}]
        records = MODULE.compare(previous, current)
        self.assertEqual("content_changed", records[0]["disposition"])


if __name__ == "__main__":
    unittest.main()
