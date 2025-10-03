package helpers

import (
	// "fmt"
	"fmt"
	"strings"
)

func GetDockerFile(server string) string {
	switch server {
	case "next":
		dockerfile := "" +
			"FROM node:18-alpine\n\n" +
			"# Set working directory\n" +
			"WORKDIR /app\n\n" +
			"# Install dependencies separately for caching\n" +
			"COPY package*.json ./\n" +
			"RUN npm install\n\n" +
			"# Copy the rest of the project\n" +
			"COPY . .\n\n" +
			"# Expose Next.js dev server port\n" +
			"EXPOSE 3000\n\n" +
			"# Default command (start dev server with hot reload)\n" +
			"CMD [\"npm\", \"run\", \"dev\"]\n"
		return dockerfile
	case "nodejs":
		// dockerfile := "FROM node:18-alpine\n\nWORKDIR /app\n\nCOPY package*.json ./\nRUN npm install --production\n\nCOPY . .\n\nEXPOSE 3000\n\nCMD [\"npm\", \"start\"]\n"
		dockerfile := "" +
			"FROM node:18-alpine\n\n" +
			"WORKDIR /app\n\n" +
			"COPY package*.json ./\n" +
			"RUN npm install --production\n\n" +
			"COPY . .\n\n" +
			"EXPOSE 3000\n\n" +
			"CMD [\"npm\", \"start\"]\n"
		return dockerfile
	default:
		return ""
	}
}

func GetDockerCompose(serverPort int, appPort int, cluster string) string {
	compose := "services:\n" +
		"  cluster896285213382735-server:\n" +
		"    build:\n" +
		"      context: ./server\n" +
		"    image: cluster896285213382735-server:latest\n" +
		"    container_name: cluster896285213382735-server\n" +
		"    ports:\n" +
		"      - \"serverPort846728457424:5001\"\n" +
		"    environment:\n" +
		"      - PORT=5001\n" +
		"      - NODE_ENV=development\n" +
		"      - CORS_ORIGIN=http://cluster896285213382735-app:3005\n" +
		"      - RATE_LIMIT_WINDOW_MS=900000\n" +
		"      - RATE_LIMIT_MAX_REQUESTS=100\n" +
		"    restart: \"no\"\n\n" +
		"  cluster896285213382735-app:\n" +
		"    build:\n" +
		"      context: ./app\n" +
		"    image: cluster896285213382735-app:latest\n" +
		"    container_name: cluster896285213382735-app\n" +
		"    ports:\n" +
		"      - \"appPort432576459289:3005\"\n" +
		"    environment:\n" +
		"      - NODE_ENV=development\n" +
		"      - NEXT_PUBLIC_API_URL=http://cluster896285213382735-server:5001\n" +
		"      - PORT=3005\n" +
		"    depends_on:\n" +
		"      - cluster896285213382735-server\n" +
		"    restart: \"no\"\n"

	// compose = strings.ReplaceAll(compose, "serverPort846728457424", fmt.Sprintf("%d", serverPort))
	compose = strings.ReplaceAll(compose, "cluster896285213382735", cluster)
	compose = strings.ReplaceAll(compose, "serverPort846728457424", fmt.Sprintf("%d", serverPort))
	compose = strings.ReplaceAll(compose, "appPort432576459289", fmt.Sprintf("%d", appPort))
	return compose
}
