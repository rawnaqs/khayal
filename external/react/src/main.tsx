import React from "react";
import ReactDOM from "react-dom/client";
import App from "./App";
import "./index.css";
import { STORAGE_KEYS } from "./lib/constants";

// Register service worker
if ("serviceWorker" in navigator) {
  window.addEventListener("load", () => {
    navigator.serviceWorker
      .register("/sw.js")
      .then((registration) => {
        console.log("SW registered:", registration.scope);

        // Listen for updates
        registration.addEventListener("updatefound", () => {
          const newWorker = registration.installing;
          if (newWorker) {
            newWorker.addEventListener("statechange", () => {
              if (newWorker.state === "activated") {
                console.log("SW activated");
              }
            });
          }
        });

        // Request background sync permission
        if ("sync" in registration) {
          (registration as any).sync
            .register("sync-offline-captures")
            .catch(() => {
              // Background sync not supported or permission denied
            });
        }
      })
      .catch((err) => {
        console.log("SW registration failed:", err);
      });
  });
}

if ((window.navigator as any).standalone === true) {
  document.documentElement.classList.add("pwa-standalone");
}

// Setup offline sync if connected
const token = localStorage.getItem(STORAGE_KEYS.TOKEN);
const host = localStorage.getItem(STORAGE_KEYS.HOST);
if (token && host) {
  import("./lib/offline").then(({ setupOfflineSync }) => {
    setupOfflineSync(host, token);
  });
}

ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>,
);
