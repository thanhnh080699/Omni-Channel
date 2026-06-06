import type { Config } from "tailwindcss";

const config: Config = {
  content: ["./app/**/*.{ts,tsx}", "./components/**/*.{ts,tsx}", "./lib/**/*.{ts,tsx}"],
  theme: {
    extend: {
      colors: {
        shell: "#f6f7f9",
        panel: "#ffffff",
        line: "#dde3ea",
        ink: "#17212b",
        muted: "#64748b",
        accent: "#1f93ff",
        success: "#0f9f6e",
        warning: "#b7791f",
        danger: "#dc2626",
      },
      boxShadow: {
        soft: "0 1px 2px rgba(15, 23, 42, 0.06)",
      },
    },
  },
  plugins: [],
};

export default config;
