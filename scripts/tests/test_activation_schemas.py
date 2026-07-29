import json
import unittest
from pathlib import Path

from jsonschema import Draft202012Validator
from referencing import Registry, Resource


ROOT = Path(__file__).resolve().parents[2]
SCHEMAS = (
    "accounting-professional-decision.schema.json",
    "company-activation.schema.json",
    "company-research-profile.schema.json",
    "peer-lane.schema.json",
    "metric-comparability-request.schema.json",
    "metric-comparability-receipt.schema.json",
    "comparison-bundle.schema.json",
    "typed-abstention.schema.json",
    "standalone-journey-suite.schema.json",
    "peer-journey-suite.schema.json",
    "technology20-financial-summary.schema.json",
    "technology20-peer-evaluation.schema.json",
    "technology20-pair-population.schema.json",
)


class ActivationSchemaTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls) -> None:
        cls.schemas = {
            name: json.loads((ROOT / "contracts" / name).read_text(encoding="utf-8"))
            for name in SCHEMAS
        }
        cls.registry = Registry()
        for schema in cls.schemas.values():
            cls.registry = cls.registry.with_resource(
                schema["$id"], Resource.from_contents(schema)
            )

    def test_sprint32_schemas_are_valid_draft_2020_12(self) -> None:
        for name in SCHEMAS:
            with self.subTest(schema=name):
                payload = json.loads((ROOT / "contracts" / name).read_text(encoding="utf-8"))
                self.assertEqual(payload["$schema"], "https://json-schema.org/draft/2020-12/schema")
                Draft202012Validator.check_schema(payload)

    def test_sprint32_schema_ids_are_unique(self) -> None:
        identifiers = []
        for name in SCHEMAS:
            payload = json.loads((ROOT / "contracts" / name).read_text(encoding="utf-8"))
            identifiers.append(payload["$id"])
        self.assertEqual(len(identifiers), len(set(identifiers)))

    def test_public_sprint32_populations_match_portable_schemas(self) -> None:
        cases = (
            ("standalone-journey-suite.schema.json", "technology20-standalone-development.json"),
            (
                "standalone-journey-suite.schema.json",
                "technology20-standalone-domain-augmentation.json",
            ),
            ("peer-journey-suite.schema.json", "technology20-peer-development.json"),
        )
        for schema_name, fixture_name in cases:
            with self.subTest(fixture=fixture_name):
                schema = json.loads((ROOT / "contracts" / schema_name).read_text(encoding="utf-8"))
                fixture = json.loads(
                    (ROOT / "fixtures" / "productscope" / fixture_name).read_text(encoding="utf-8")
                )
                errors = sorted(Draft202012Validator(schema).iter_errors(fixture), key=lambda item: list(item.path))
                self.assertEqual(errors, [])

    def test_public_financial_and_peer_api_fixtures_match_portable_schemas(self) -> None:
        cases = (
            (
                "technology20-financial-summary.schema.json",
                "technology20-financial-summary.json",
            ),
            (
                "technology20-peer-evaluation.schema.json",
                "technology20-peer-evaluation.json",
            ),
        )
        for schema_name, fixture_name in cases:
            with self.subTest(fixture=fixture_name):
                fixture = json.loads(
                    (ROOT / "fixtures" / "productscope" / fixture_name).read_text(
                        encoding="utf-8"
                    )
                )
                validator = Draft202012Validator(
                    self.schemas[schema_name], registry=self.registry
                )
                errors = sorted(
                    validator.iter_errors(fixture), key=lambda item: list(item.path)
                )
                self.assertEqual(errors, [])

    def test_professional_decision_fixture_matches_portable_schema(self) -> None:
        fixture = json.loads(
            (
                ROOT
                / "fixtures"
                / "productscope"
                / "accounting-authority"
                / "technology20-accounting-professional-decision.json"
            ).read_text(encoding="utf-8")
        )
        validator = Draft202012Validator(
            self.schemas["accounting-professional-decision.schema.json"],
            registry=self.registry,
        )
        errors = sorted(
            validator.iter_errors(fixture), key=lambda item: list(item.path)
        )
        self.assertEqual(errors, [])


if __name__ == "__main__":
    unittest.main()
