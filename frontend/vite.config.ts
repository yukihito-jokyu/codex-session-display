import path from "node:path";
import react from "@vitejs/plugin-react";
import { defineConfig } from "vite";
import istanbul from "vite-plugin-istanbul";

// https://vitejs.dev/config/
export default defineConfig({
	plugins: [
		react(),
		istanbul({
			include: ["src/**/*.ts", "src/**/*.tsx"],
			exclude: ["node_modules", "tests", "wailsjs"],
			extension: [".js", ".ts", ".tsx"],
			requireEnv: true,
		}),
	],
	resolve: {
		alias: {
			wailsjs: path.resolve(__dirname, "./wailsjs"),
		},
	},
});
