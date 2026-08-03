export interface ConnectorAuthConfig {
  clientId?: string;
  secret?: string;
  env?: string;
  [key: string]: any;
}

export interface Connector {
  id: string;
  name: string;
  type: "mcp" | "rest" | "graphql";
  endpoint: string;
  authConfig?: ConnectorAuthConfig;
  status?: "connected" | "disconnected" | "error";
  description?: string;
}
