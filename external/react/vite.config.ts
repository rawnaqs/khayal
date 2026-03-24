import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import { VitePWA } from "vite-plugin-pwa";
import path from "path";

export default defineConfig({
  plugins: [
    react(),
    VitePWA({
      registerType: "autoUpdate",
      includeAssets: ["icon-192.png", "icon-512.png"],
      manifest: {
        name: "Khayal",
        short_name: "khayal",
        description: "Personal knowledge vault",
        start_url: "/",
        display: "standalone",
        orientation: "portrait",
        background_color: "#0f0f0f",
        theme_color: "#C9933A",
        icons: [
          { src: "/icon-192.png", sizes: "192x192", type: "image/png" },
          { src: "/icon-512.png", sizes: "512x512", type: "image/png" },
        ],
      },
      workbox: {
        globPatterns: ["**/*.{js,css,html,ico,png,svg}"],
        runtimeCaching: [
          {
            // App shell — cache first
            urlPattern: /^https?:\/\/.*\.(js|css|html|ico|png|svg)$/,
            handler: "CacheFirst",
            options: {
              cacheName: "khayal-shell",
              expiration: {
                maxEntries: 50,
                maxAgeSeconds: 30 * 24 * 60 * 60, // 30 days
              },
            },
          },
          {
            // Health — network first
            urlPattern: /\/v1\/health/,
            handler: "NetworkFirst",
            options: {
              cacheName: "khayal-health",
              expiration: {
                maxEntries: 1,
                maxAgeSeconds: 60, // 1 minute
              },
            },
          },
          {
            // Stats — stale while revalidate
            urlPattern: /\/v1\/stats/,
            handler: "StaleWhileRevalidate",
            options: {
              cacheName: "khayal-stats",
              expiration: {
                maxEntries: 1,
                maxAgeSeconds: 60, // 1 minute
              },
            },
          },
          {
            // Search — network first
            urlPattern: /\/v1\/search/,
            handler: "NetworkFirst",
            options: {
              cacheName: "khayal-search",
              expiration: {
                maxEntries: 20,
                maxAgeSeconds: 5 * 60, // 5 minutes
              },
            },
          },
          {
            // Queue — network first
            urlPattern: /\/v1\/queue/,
            handler: "NetworkFirst",
            options: {
              cacheName: "khayal-queue",
              expiration: {
                maxEntries: 10,
                maxAgeSeconds: 5 * 60, // 5 minutes
              },
            },
          },
          {
            // Capture — network only (no cache)
            urlPattern: /\/v1\/capture/,
            handler: "NetworkOnly",
          },
        ],
      },
    }),
  ],
  base: "/",
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
  build: {
    outDir: "../../internal/api/ui/static",
    emptyOutDir: true,
  },
  server: {
    port: 5173,
    proxy: {
      "/v1": {
        target: "http://localhost:1133",
        changeOrigin: true,
      },
    },
  },
});
