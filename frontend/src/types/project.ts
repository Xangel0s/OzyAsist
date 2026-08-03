export interface ProjectFile {
  path: string;
  name: string;
  size?: number;
  isDir?: boolean;
}

export interface Project {
  id: string;
  name: string;
  path: string;
  description?: string;
  createdAt?: string;
  updatedAt?: string;
  files?: ProjectFile[];
}
