import { create } from "zustand";
import { persist } from "zustand/middleware";
import { api, type ProjectDTO } from "../services/api";
import { useChatStore } from "./chatStore";

export interface ProjectFile {
  name: string;
  type: "file" | "folder";
  path?: string;
  size?: number;
  children?: ProjectFile[];
}

export interface Project {
  id: string;
  name: string;
  description: string;
  instructions: string;
  rootPath?: string;
  permissionLevel?: string;
  files?: ProjectFile[];
  chatIds?: string[];
  createdAt: string;
}

interface ProjectsState {
  projects: Project[];
  activeProjectId: string | null;
  loading: boolean;
  setActiveProject: (id: string | null) => void;
  addProject: (project: Project) => void;
  loadProjects: () => Promise<void>;
  createProject: (name: string, rootPath?: string, instructionsMd?: string, permissionLevel?: string) => Promise<string | null>;
  deleteProject: (id: string) => Promise<void>;
  updateProject: (id: string, data: { name?: string; rootPath?: string; instructionsMd?: string }) => Promise<void>;
}

const dtoToProject = (dto: ProjectDTO): Project => ({
  id: dto.id,
  name: dto.name,
  description: dto.rootPath ? `Ruta: ${dto.rootPath}` : "Proyecto sin ruta",
  instructions: dto.instructionsMd || "",
  rootPath: dto.rootPath,
  permissionLevel: dto.permissionLevel || "sandboxed",
  createdAt: dto.createdAt,
});

export const useProjectsStore = create<ProjectsState>()(
  persist(
    (set) => ({
  projects: [],
  activeProjectId: null,
  loading: true,

  setActiveProject: (id) => set({ activeProjectId: id }),

  addProject: (project) =>
    set((s) => ({ projects: [...s.projects, project] })),

  loadProjects: async () => {
    set({ loading: true });
    try {
      const dtos = await api.projects.list();
      set({ projects: dtos.map(dtoToProject), loading: false });
    } catch (e) {
      console.error("loadProjects failed:", e);
      set({ loading: false });
    }
  },

  createProject: async (name, rootPath, instructionsMd, permissionLevel) => {
    try {
      const dto = await api.projects.create({ name, rootPath, instructionsMd, permissionLevel });
      const project = dtoToProject(dto);
      set((s) => ({ projects: [...s.projects, project] }));
      return project.id;
    } catch (e) {
      console.error("createProject failed:", e);
      return null;
    }
  },

  deleteProject: async (id) => {
    try {
      await api.projects.delete(id);
      set((s) => ({
        projects: s.projects.filter((p) => p.id !== id),
        activeProjectId: s.activeProjectId === id ? null : s.activeProjectId,
      }));
      // Clean up related chats
      const { chats, updateChatTitle } = useChatStore.getState();
      for (const chat of chats) {
        if (chat.mode === "code" && chat.title.includes("(Proyecto")) {
          updateChatTitle(chat.id, chat.title.replace(/\(Proyecto[^)]*\)/, ""));
        }
      }
    } catch (e) {
      console.error("deleteProject failed:", e);
    }
  },

  updateProject: async (id, data) => {
    try {
      const dto = await api.projects.update(id, data);
      set((s) => ({
        projects: s.projects.map((p) => (p.id === id ? dtoToProject(dto) : p)),
      }));
    } catch (e) {
      console.error("updateProject failed:", e);
    }
  },
}),
    {
      name: "ozy-projects",
      partialize: (state) => ({
        projects: state.projects,
        activeProjectId: state.activeProjectId,
      }),
    },
  ),
);

// Init: load projects from backend
useProjectsStore.getState().loadProjects();
