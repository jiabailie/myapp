# Use the official Node.js image
FROM node:lts-alpine

# Set the working directory
WORKDIR /app

# Copy package.json and package-lock.json
COPY package*.json ./

# Install dependencies
RUN npm install

# Copy the rest of the application code
COPY . .

# Build the React app
RUN npm run build

# Install `serve` to serve the build directory
RUN npm install -g serve

# Expose port 3000 to the outside world
EXPOSE 3000

# Serve the build directory on port 3000
CMD ["serve", "-s", "build"]
