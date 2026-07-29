from __future__ import annotations

import re
import unittest
from pathlib import Path


ROOT = Path(__file__).resolve().parents[2]
STYLES = (ROOT / "web/src/styles.css").read_text(encoding="utf-8")
COMPACT_STATUS = (ROOT / "web/src/components/CompactRunStatus.tsx").read_text(
    encoding="utf-8"
)
INSIGHT_PANEL = (ROOT / "web/src/components/InsightPanel.tsx").read_text(encoding="utf-8")


class WebAccessibilityContractTests(unittest.TestCase):
    def test_critical_text_pairs_meet_wcag_aa_contrast(self) -> None:
        pairs = {
            "body text": (variable("--ink"), variable("--paper")),
            "muted text on the darkest page field": (variable("--muted"), "#e8e3d7"),
            "primary action": ("#ffffff", variable("--green")),
            "sidebar navigation index": (
                declaration(".section-links button span", "color"),
                declaration(".side-nav", "background"),
            ),
            "warning classification": (
                variable("--copper"),
                declaration(".warning-ledger", "background"),
            ),
            "persistent product warning": (
                declaration(".site-footer", "color"),
                "#e8e3d7",
            ),
        }

        failures = []
        for label, (foreground, background) in pairs.items():
            ratio = contrast_ratio(foreground, background)
            if ratio < 4.5:
                failures.append(
                    f"{label}: {foreground} on {background} = {ratio:.2f}:1"
                )
        self.assertEqual(failures, [])

    def test_operational_states_have_non_color_language(self) -> None:
        for label in (
            "Execution detail is reconnecting",
            "Research stopped safely",
            "Bounded answer ready",
            "No partial replacement was released",
            "Only the evidence-supported subset was released",
        ):
            self.assertIn(label, COMPACT_STATUS)

        fixture = (ROOT / "fixtures/workspace/golden-case.json").read_text()
        self.assertIn('"kind": "missing_evidence"', fixture)
        self.assertIn('"kind": "uncertainty"', fixture)
        self.assertIn('warning.kind.replaceAll("_", " ")', INSIGHT_PANEL)


def variable(name: str) -> str:
    match = re.search(rf"{re.escape(name)}:\s*(#[0-9a-fA-F]{{6}})", STYLES)
    if not match:
        raise AssertionError(f"CSS variable {name} not found")
    return match.group(1)


def declaration(selector: str, property_name: str) -> str:
    block = re.search(rf"{re.escape(selector)}\s*\{{([^}}]+)\}}", STYLES)
    if not block:
        raise AssertionError(f"CSS selector {selector} not found")
    value = re.search(
        rf"(?:^|;)\s*{re.escape(property_name)}:\s*(#[0-9a-fA-F]{{6}})",
        block.group(1),
    )
    if not value:
        raise AssertionError(f"{property_name} not found in {selector}")
    return value.group(1)


def contrast_ratio(foreground: str, background: str) -> float:
    light = max(relative_luminance(foreground), relative_luminance(background))
    dark = min(relative_luminance(foreground), relative_luminance(background))
    return (light + 0.05) / (dark + 0.05)


def relative_luminance(color: str) -> float:
    channels = [int(color[index : index + 2], 16) / 255 for index in (1, 3, 5)]
    linear = [
        channel / 12.92
        if channel <= 0.03928
        else ((channel + 0.055) / 1.055) ** 2.4
        for channel in channels
    ]
    return 0.2126 * linear[0] + 0.7152 * linear[1] + 0.0722 * linear[2]


if __name__ == "__main__":
    unittest.main()
