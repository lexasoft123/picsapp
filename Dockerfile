# Stage 1: Build React frontend
FROM node:18-alpine AS frontend-builder
WORKDIR /app
COPY package*.json ./
RUN npm ci
COPY src/ ./src/
COPY public/ ./public/
RUN npm run build

# Stage 2: Build Go backend
FROM golang:1.21-alpine AS backend-builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY main.go database.go ./
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o picsapp main.go database.go

# Stage 3: Runtime image
FROM alpine:latest
RUN apk --no-cache add ca-certificates sqlite wget
WORKDIR /app

# Copy built frontend
COPY --from=frontend-builder /app/build ./build

# Copy Go binary
COPY --from=backend-builder /app/picsapp .

# Create directories for uploads and database
RUN mkdir -p uploads/original

# Expose port
EXPOSE 8080

# Set environment variable
ENV PORT=8080

# Run the application
CMD ["./picsapp"]

