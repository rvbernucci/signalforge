from __future__ import annotations

import importlib.util
from pathlib import Path
import unittest
from unittest import mock


MODULE_PATH = Path(__file__).resolve().parents[1] / "ir_collect.py"
SPEC = importlib.util.spec_from_file_location("ir_collect", MODULE_PATH)
assert SPEC and SPEC.loader
MODULE = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(MODULE)


class IRCollectorTests(unittest.TestCase):
    def test_canonical_uri_drops_tracking_and_fragment(self) -> None:
        self.assertEqual(
            "https://ir.example.com/report.pdf?id=7",
            MODULE.canonical_uri("https://IR.EXAMPLE.COM/report.pdf?id=7&utm_source=x#page=2"),
        )

    def test_year_filter_keeps_undated_and_current_window(self) -> None:
        self.assertTrue(MODULE.eligible_year("https://example.com/history", "", 2021))
        self.assertTrue(MODULE.eligible_year("https://example.com/2025/report", "", 2021))
        self.assertFalse(MODULE.eligible_year("https://example.com/2019/report", "", 2021))

    def test_classification_is_conservative(self) -> None:
        self.assertEqual(
            ("prepared_remarks", "C", False),
            MODULE.classify("https://example.com/q1.pdf", "Prepared Remarks", ""),
        )
        self.assertEqual(
            ("corporate_profile_and_history", "E", True),
            MODULE.classify("https://example.com/company-history", "", ""),
        )
        self.assertEqual(
            ("official_strategy_or_risk_update", "C", False),
            MODULE.classify("https://example.com/investors", "", "Investor Relations"),
        )
        self.assertEqual(
            ("capital_allocation_update", "C", False),
            MODULE.classify("https://example.com/news", "Quarterly dividend", ""),
        )
        self.assertEqual(
            ("governance_document", "D", False),
            MODULE.classify("https://example.com/policy", "Partner Code of Conduct", ""),
        )

    def test_generic_investor_path_does_not_make_a_page_material(self) -> None:
        self.assertFalse(MODULE.is_material_candidate("https://example.com/investor/faq", "FAQ"))
        self.assertTrue(MODULE.is_material_candidate("https://example.com/investor/quarterly-results", "Results"))

    def test_material_filter_ignores_host_and_navigation_noise(self) -> None:
        self.assertFalse(MODULE.is_material_candidate(
            "https://www.aboutamazon.com/news/company-news", "Company news"
        ))
        self.assertFalse(MODULE.is_material_candidate(
            "https://www.microsoft.com/en-us/privacy?icid=Company_Privacy", "Privacy"
        ))
        self.assertFalse(MODULE.is_material_candidate(
            "https://ir.example.com/annual-reports", "Annual Reports"
        ))
        self.assertTrue(MODULE.is_material_candidate(
            "https://ir.example.com/about/history", "Our history"
        ))
        self.assertTrue(MODULE.is_material_candidate(
            "https://ir.example.com/shareholder-letters", "Annual reports and shareholder letters"
        ))
        self.assertFalse(MODULE.is_material_candidate(
            "https://blogs.example.com/", "Official corporate blog"
        ))
        self.assertTrue(MODULE.is_material_candidate(
            "https://www.example.com/en-us/about-company/history/",
            "About Us, History, Jobs, Shopping Cart",
        ))
        self.assertTrue(MODULE.is_material_candidate(
            "https://news.example.com/2026/update", "Quarterly dividend declared"
        ))

    def test_retry_delay_is_bounded(self) -> None:
        self.assertEqual(3.0, MODULE.retry_delay("3", 1))
        self.assertEqual(1.0, MODULE.retry_delay(None, 1))
        self.assertEqual(4.0, MODULE.retry_delay(None, 9))
        self.assertEqual(30.0, MODULE.retry_delay("999", 1))

    @mock.patch.object(MODULE.ir_discover, "fetch")
    def test_robots_guard_fails_closed_on_external_redirect(self, fetch: mock.Mock) -> None:
        fetch.return_value = {
            "status_code": 200,
            "final_uri": "https://attacker.test/robots.txt",
            "payload": b"User-agent: *\nAllow: /\n",
            "elapsed_ms": 1,
        }
        guard = MODULE.RobotsGuard("SignalForge test", 1, MODULE.HostBudget(1000))
        allowed, observation = guard.check("https://ir.example.com/report", "sec-cik:0000000001", ["example.com"])
        self.assertFalse(allowed)
        self.assertEqual("disallowed", observation["disposition"])


if __name__ == "__main__":
    unittest.main()
