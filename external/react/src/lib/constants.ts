// VERSION
declare const __APP_VERSION__: string;
export const APP_VERSION = __APP_VERSION__;

// GitHub
export const GITHUB_RELEASES_URL = "https://github.com/rawnaqs/khayal/releases/latest";

// localStorage keys
export const STORAGE_KEYS = {
  TOKEN: "khayal_token",
  HOST: "khayal_host",
  RECENT_SEARCHES: "khayal-recent-searches",
} as const;

// Search suggestions (shown in idle state)
export const SEARCH_SUGGESTIONS = [
  "people",
  "payments",
  "this week",
  "ideas",
  "decisions",
  "meetings",
];

// Processing steps by capture type
export const PROCESSING_STEPS: Record<string, string[]> = {
  text: ["saved", "tagging", "summarizing", "writing"],
  image: ["saved", "describing", "tagging", "writing"],
  article: ["saved", "extracting", "summarizing", "writing"],
};

// Limits
export const LIMITS = {
  SEARCH_RESULTS: 20,
  QUEUE_JOBS: 50,
  RECENT_SEARCHES: 10,
  DONE_JOBS_SHOWN: 5,
  TAGS_HERO: 3,
  TAGS_COMPACT: 2,
  HERO_SCORE_THRESHOLD: 0.9,
} as const;

// Timeouts (ms)
export const TIMEOUTS = {
  CAPTURE_DISMISS: 3500,
  STATS_POLL: 60000,
  SERVER_STATUS_POLL: 30000,
} as const;

// Greeting messages with hour thresholds
export const GREETINGS = [
  { maxHour: 5, text: "late night thoughts?" },
  { maxHour: 12, text: "good morning" },
  { maxHour: 17, text: "good afternoon" },
  { maxHour: 21, text: "good evening" },
  { maxHour: 24, text: "late night thoughts?" },
] as const;

// Type filters for search
export const TYPE_FILTERS = ["all", "text", "article", "image"] as const;

// Search modes
export const SEARCH_MODES = ["hybrid", "keyword", "semantic"] as const;
