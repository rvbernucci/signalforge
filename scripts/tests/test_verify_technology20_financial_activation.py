import importlib.util
import unittest
from decimal import Decimal
from pathlib import Path


ROOT = Path(__file__).resolve().parents[2]
SPEC = importlib.util.spec_from_file_location(
    "verify_technology20_financial_activation",
    ROOT / "scripts" / "verify_technology20_financial_activation.py",
)
MODULE = importlib.util.module_from_spec(SPEC)
assert SPEC.loader is not None
SPEC.loader.exec_module(MODULE)


class Technology20FinancialReferenceTests(unittest.TestCase):
    def test_recomputes_simple_fcf_without_promoting_fcff(self) -> None:
        receipt = {
            "operation_id": "financial.free_cash_flow",
            "normalized_inputs": [
                {"input_id": "operating_cash_flow", "quantity": {"value": "30"}},
                {"input_id": "capital_expenditure", "quantity": {"value": "10"}},
            ],
        }
        self.assertEqual(
            MODULE.independently_recompute(receipt),
            {"free_cash_flow": Decimal("20")},
        )

    def test_recomputes_quality_of_earnings(self) -> None:
        receipt = {
            "operation_id": "financial.quality_of_earnings",
            "normalized_inputs": [
                {"input_id": "operating_cash_flow", "quantity": {"value": "30"}},
                {"input_id": "net_income", "quantity": {"value": "20"}},
            ],
        }
        self.assertEqual(
            MODULE.independently_recompute(receipt),
            {"accrual_gap": Decimal("10"), "cash_conversion": Decimal("1.5")},
        )

    def test_context_only_receipt_cannot_claim_pair_ranking(self) -> None:
        receipt = {
            "receipt_id": "receipt-1",
            "operation_id": "financial.free_cash_flow",
            "normalized_inputs": [
                {
                    "input_id": "operating_cash_flow",
                    "quantity": {"value": "30"},
                },
                {
                    "input_id": "capital_expenditure",
                    "quantity": {"value": "10"},
                },
            ],
            "outputs": [
                {
                    "output_id": "free_cash_flow",
                    "quantity": {"value": "20", "unit": "currency"},
                }
            ],
            "invariant_results": [{"passed": True}],
            "evidence_refs": ["fact-capex", "fact-ocf"],
        }
        authority = {
            "receipt_id": "receipt-1",
            "operation_id": "financial.free_cash_flow",
            "output_class": "context_only",
            "product_label": "residual cash proxy",
            "accounting_perimeter_signature": (
                "capital_expenditure=company_reported_cash_capital_expenditures;"
                "operating_cash_flow=consolidated_periodic_filing"
            ),
            "pair_ranking_eligible": True,
            "inputs": [
                {
                    "input_id": "capital_expenditure",
                    "source_fact_ids": ["fact-capex"],
                    "context_only": True,
                },
                {
                    "input_id": "operating_cash_flow",
                    "source_fact_ids": ["fact-ocf"],
                    "context_only": False,
                },
            ],
        }
        self.assertIn(
            "company:financial.free_cash_flow:context_only_ranking_escape",
            MODULE.verify_receipt(
                "company",
                receipt,
                authority,
                "context_only",
            ),
        )


if __name__ == "__main__":
    unittest.main()
