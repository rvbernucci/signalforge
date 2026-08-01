import { afterEach, describe, expect, it, vi } from "vitest";
import { getCase, listCases } from "./api";

afterEach(() => {
  vi.restoreAllMocks();
});

describe("case API cache boundary", () => {
  it("always refreshes the local case index", async () => {
    const fetchMock = vi.spyOn(globalThis, "fetch").mockResolvedValue(
      new Response(JSON.stringify({ cases: [] }), {
        status: 200,
        headers: { "Content-Type": "application/json" }
      })
    );

    await expect(listCases()).resolves.toEqual([]);
    expect(fetchMock).toHaveBeenCalledWith("/api/v1/cases", { cache: "no-store" });
  });

  it("always refreshes an individual retained case", async () => {
    const payload = { case_id: "case-1" };
    const fetchMock = vi.spyOn(globalThis, "fetch").mockResolvedValue(
      new Response(JSON.stringify(payload), {
        status: 200,
        headers: { "Content-Type": "application/json" }
      })
    );

    await expect(getCase("case 1")).resolves.toEqual(payload);
    expect(fetchMock).toHaveBeenCalledWith("/api/v1/cases/case%201", { cache: "no-store" });
  });
});
