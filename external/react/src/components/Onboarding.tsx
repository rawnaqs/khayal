import { useState } from "react";
import { motion } from "framer-motion";
import { STORAGE_KEYS } from "@/lib/constants";

interface OnboardingProps {
  onComplete: () => void;
}

export function Onboarding({ onComplete }: OnboardingProps) {
  const [token, setToken] = useState("");
  const [error, setError] = useState("");
  const [testing, setTesting] = useState(false);

  const testConnection = async () => {
    if (!token) {
      setError("enter your token to connect");
      return;
    }

    setTesting(true);
    setError("");

    try {
      const host = window.location.origin;
      const response = await fetch(`${host}/v1/health`, {
        headers: { "X-Khayal-Token": token },
      });

      if (!response.ok) {
        throw new Error("Invalid token");
      }

      localStorage.setItem(STORAGE_KEYS.HOST, host);
      localStorage.setItem(STORAGE_KEYS.TOKEN, token);
      onComplete();
    } catch {
      setError("cannot connect — check the token and try again");
    } finally {
      setTesting(false);
    }
  };

  return (
    <div className="flex flex-col items-center justify-center h-screen p-8" style={{ background: "#070707" }}>
      <motion.div
        initial={{ opacity: 0, scale: 0.95 }}
        animate={{ opacity: 1, scale: 1 }}
        transition={{ duration: 0.5, ease: "easeOut" }}
        className="flex flex-col items-center gap-6 w-full max-w-sm"
      >
        <motion.img
          src="/icon.svg"
          alt="khayal"
          className="w-16 h-16"
          initial={{ scale: 0.8, opacity: 0 }}
          animate={{ scale: 1, opacity: 1 }}
          transition={{ delay: 0.1, duration: 0.5, ease: "easeOut" }}
        />

        <div className="text-center">
          <h1 className="text-2xl font-bold tracking-tight" style={{ fontFamily: "'Bricolage Grotesque', sans-serif", color: "#f5f5f5" }}>
            khayal
          </h1>
          <p className="mt-1 text-sm" style={{ fontFamily: "'IBM Plex Mono', monospace", color: "rgba(245,245,245,0.4)", fontSize: 11 }}>
            your private treasury of thought
          </p>
        </div>

        <div className="w-full p-6 rounded-2xl" style={{ background: "#141414", border: "1px solid rgba(255,255,255,0.08)" }}>
          <label
            htmlFor="khayal-token"
            className="block mb-2 tracking-wider"
            style={{ fontFamily: "'IBM Plex Mono', monospace", fontSize: 9, color: "rgba(245,245,245,0.2)", textTransform: "uppercase" }}
          >
            token
          </label>
          <input
            id="khayal-token"
            type="text"
            placeholder="a3f9c2e..."
            value={token}
            onChange={(e) => setToken(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter") testConnection();
            }}
            className="w-full px-3 py-3 rounded-xl text-base outline-none transition-all duration-200"
            style={{
              fontFamily: "'IBM Plex Mono', monospace",
              fontSize: 16,
              background: "#0d0d0d",
              border: "1px solid rgba(255,255,255,0.09)",
              color: "#f5f5f5",
            }}
            onFocus={(e) => {
              e.currentTarget.style.borderColor = "rgba(201,147,58,0.3)";
              e.currentTarget.style.boxShadow = "0 0 0 3px rgba(201,147,58,0.06)";
            }}
            onBlur={(e) => {
              e.currentTarget.style.borderColor = "rgba(255,255,255,0.09)";
              e.currentTarget.style.boxShadow = "none";
            }}
          />
          <p className="mt-1 text-xs" style={{ fontFamily: "'IBM Plex Mono', monospace", fontSize: 9, color: "rgba(245,245,245,0.2)" }}>
            find it in ~/.config/khayal/config.yaml
          </p>

          {error && (
            <motion.p
              initial={{ opacity: 0, y: -4 }}
              animate={{ opacity: 1, y: 0 }}
              className="mt-3 text-sm text-center"
              style={{ fontFamily: "'IBM Plex Mono', monospace", fontSize: 11, color: "#ff4d4d" }}
            >
              {error}
            </motion.p>
          )}

          <motion.div whileTap={{ scale: 0.98 }} className="mt-4">
            <button
              onClick={testConnection}
              disabled={testing}
              className="w-full h-12 rounded-xl font-semibold tracking-wide transition-all duration-200"
              style={{
                fontFamily: "'IBM Plex Mono', monospace",
                fontSize: 13,
                background: testing
                  ? "rgba(201,147,58,0.2)"
                  : "linear-gradient(135deg, #c9933a 0%, #a67830 100%)",
                color: testing ? "rgba(201,147,58,0.5)" : "#000",
                border: testing ? "1px solid rgba(201,147,58,0.2)" : "none",
                cursor: testing ? "default" : "pointer",
              }}
            >
              {testing ? "connecting..." : "enter the lamp room"}
            </button>
          </motion.div>
        </div>
      </motion.div>
    </div>
  );
}
