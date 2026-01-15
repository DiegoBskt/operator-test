# Build stage
FROM registry.access.redhat.com/ubi9/nodejs-18:latest as build

USER root
WORKDIR /app

# Copy package files
COPY package.json yarn.lock ./

# Install dependencies
RUN npm install -g yarn && yarn install --frozen-lockfile

# Copy source
COPY . .

# Build the plugin
RUN yarn build

# Production stage
FROM registry.access.redhat.com/ubi9/nginx-122:latest

# Copy built assets to nginx
COPY --from=build /app/dist /opt/app-root/src

# Custom nginx config for console plugin
COPY nginx.conf /etc/nginx/nginx.conf

# Labels
LABEL name="cluster-assessment-plugin" \
    vendor="Community" \
    summary="OpenShift Console Plugin for Cluster Assessment Operator" \
    description="Dynamic console plugin providing UI for viewing cluster assessments"

EXPOSE 9001

CMD ["nginx", "-g", "daemon off;"]
