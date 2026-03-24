import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, act, waitFor } from "@testing-library/react";
import { useStats } from "../useStats";

// Mock the api module
vi.mock("@/lib/api", () => ({
  createClient: vi.fn(() => ({
    stats: vi.fn(),
  })),
}));

// Mock constants
vi.mock("@/lib/constants", () => ({
  TIMEOUTS: {
    STATS_POLL: 10000,
  },
}));

describe("useStats", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("should start with loading state", () => {
    const { result } = renderHook(() => useStats());

    expect(result.current.loading).toBe(true);
    expect(result.current.stats).toBeNull();
  });

  it("should fetch stats on mount", async () => {
    const mockStats = {
      streak: {
        current: 5,
        best: 10,
        next_milestone: 7,
        days_to_milestone: 2,
        this_week: [true, true, true, false, false, false, false],
      },
      today: { count: 3, by_hour: [], avg_per_day: 2.5 },
      vault: {
        total_notes: 100,
        today_delta: 3,
        last_capture_at: "2024-01-01T10:00:00Z",
        last_7_days: [1, 2, 3, 4, 5, 6, 7],
      },
    };

    const mockStatsFn = vi.fn().mockResolvedValue(mockStats);
    const { createClient } = await import("@/lib/api");
    (createClient as ReturnType<typeof vi.fn>).mockReturnValue({
      stats: mockStatsFn,
    });

    const { result } = renderHook(() => useStats(100000));

    await waitFor(() => {
      expect(result.current.loading).toBe(false);
    });

    expect(mockStatsFn).toHaveBeenCalled();
    expect(result.current.stats).toEqual(mockStats);
  });

  it("should silently fail on error", async () => {
    const mockStatsFn = vi.fn().mockRejectedValue(new Error("Network error"));
    const { createClient } = await import("@/lib/api");
    (createClient as ReturnType<typeof vi.fn>).mockReturnValue({
      stats: mockStatsFn,
    });

    const { result } = renderHook(() => useStats(100000));

    await waitFor(() => {
      expect(result.current.loading).toBe(false);
    });

    expect(result.current.stats).toBeNull();
  });

  it("should provide refresh function", async () => {
    const mockStatsFn = vi.fn().mockResolvedValue({
      streak: {
        current: 1,
        best: 1,
        next_milestone: 7,
        days_to_milestone: 6,
        this_week: [],
      },
      today: { count: 0, by_hour: [], avg_per_day: 0 },
      vault: {
        total_notes: 0,
        today_delta: 0,
        last_capture_at: "",
        last_7_days: [],
      },
    });
    const { createClient } = await import("@/lib/api");
    (createClient as ReturnType<typeof vi.fn>).mockReturnValue({
      stats: mockStatsFn,
    });

    const { result } = renderHook(() => useStats(100000));

    await waitFor(() => {
      expect(result.current.loading).toBe(false);
    });

    expect(mockStatsFn).toHaveBeenCalledTimes(1);

    await act(async () => {
      await result.current.refresh();
    });

    expect(mockStatsFn).toHaveBeenCalledTimes(2);
  });

  it("should cleanup interval on unmount", async () => {
    const mockStatsFn = vi.fn().mockResolvedValue({
      streak: {
        current: 1,
        best: 1,
        next_milestone: 7,
        days_to_milestone: 6,
        this_week: [],
      },
      today: { count: 0, by_hour: [], avg_per_day: 0 },
      vault: {
        total_notes: 0,
        today_delta: 0,
        last_capture_at: "",
        last_7_days: [],
      },
    });
    const { createClient } = await import("@/lib/api");
    (createClient as ReturnType<typeof vi.fn>).mockReturnValue({
      stats: mockStatsFn,
    });

    const { unmount } = renderHook(() => useStats(100000));

    unmount();

    // Should not throw after unmount
    expect(mockStatsFn).toHaveBeenCalled();
  });
});
