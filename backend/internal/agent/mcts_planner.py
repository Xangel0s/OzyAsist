#!/usr/bin/env python3
"""
MCTS Planner for OzyAssist (NINE Cognitive Strategist)
High-performance Monte Carlo Tree Search (MCTS) with UCB1/PUCT for:
1. Multi-step action trajectory optimization
2. Autonomous Error Recovery & Root-Cause Backtracking
3. Blast Radius & Risk-Aware Heuristic Simulation
"""

import sys
import os
import json
import math
import time
import argparse
from typing import List, Dict, Any, Optional

class MCTSNode:
    def __init__(self, action: Dict[str, Any], parent: Optional['MCTSNode'] = None, prior: float = 1.0, depth: int = 0):
        self.action = action  # dict: {"name": str, "tool": str, "args": dict, "rationale": str, "risk": float}
        self.parent = parent
        self.children: List['MCTSNode'] = []
        self.visits = 0
        self.value = 0.0  # Cumulative reward
        self.prior = prior
        self.depth = depth

    @property
    def q_value(self) -> float:
        return self.value / (self.visits + 1e-4)

    def ucb1_score(self, c_puct: float = 1.414) -> float:
        if not self.parent:
            return 0.0
        parent_visits = max(1, self.parent.visits)
        exploration = c_puct * self.prior * (math.sqrt(parent_visits) / (1 + self.visits))
        return self.q_value + exploration

    def is_fully_expanded(self, max_candidates: int) -> bool:
        return len(self.children) >= max_candidates

class MCTSPlanner:
    def __init__(self, goal: str, context_data: str = "", error_context: Optional[Dict[str, Any]] = None, iterations: int = 60):
        self.goal = goal
        self.context_data = context_data
        self.error_context = error_context or {}
        self.iterations = iterations
        self.root = MCTSNode(action={"name": "root", "tool": "none", "args": {}, "rationale": "Initial State", "risk": 0.0})

    def _get_candidate_actions(self, current_node: MCTSNode) -> List[Dict[str, Any]]:
        """Genera acciones candidatas según si estamos en modo planificación o recuperación de error."""
        depth = current_node.depth
        has_error = bool(self.error_context.get("failed_tool") or self.error_context.get("error_message"))

        # --- MODO RECUPERACIÓN DE ERROR ---
        if has_error and depth == 0:
            failed_tool = self.error_context.get("failed_tool", "")
            err_msg = str(self.error_context.get("error_message", "")).lower()

            candidates = []
            # Rama 1: Fallback a script Python analítico en ~/.ozy/workspace
            candidates.append({
                "name": "python_workspace_fallback",
                "tool": "os_python_exec",
                "args": {"script": f"# Script de remediación para resolver: {self.goal}\n# Datos reales e inspección segura\nprint('Recuperando estado...')"},
                "rationale": "Aislar la operación en la mesa de trabajo Python 3.11 sin saturar la terminal ni depender de comandos frágiles.",
                "risk": 0.1,
                "prior": 0.95
            })

            # Rama 2: Si el fallo fue en proceso/ventana, usar WMI / ShellExecute elevado
            if "kill" in failed_tool or "close" in failed_tool or "access" in err_msg or "denegado" in err_msg or "740" in err_msg:
                candidates.append({
                    "name": "elevated_wmi_recovery",
                    "tool": "os_kill_process",
                    "args": {"force": True, "method": "wmi"},
                    "rationale": "Escalar a WMI/CIM nativo para forzar cierre con permisos administrativos eludiendo error 740.",
                    "risk": 0.4,
                    "prior": 0.90
                })

            # Rama 3: Si faltan dependencias o archivo no encontrado, explorar sistema de archivos
            if "not found" in err_msg or "no such file" in err_msg or "modulo" in err_msg or "path" in err_msg:
                candidates.append({
                    "name": "explore_and_locate_dependencies",
                    "tool": "os_find_files",
                    "args": {"name_pattern": "*.*", "max_results": 10},
                    "rationale": "Auditar rutas reales en disco para corregir la ruta antes de volver a ejecutar la acción.",
                    "risk": 0.05,
                    "prior": 0.92
                })

            # Rama 4: Consultar Grafo de Conocimiento del Sistema
            candidates.append({
                "name": "query_system_ontology",
                "tool": "query_system_knowledge",
                "args": {"query": f"recuperacion de fallo {failed_tool}", "topK": 3},
                "rationale": "Extraer del grafo ontológico el contrato exacto y herramientas co-ocurrentes para la tarea.",
                "risk": 0.0,
                "prior": 0.88
            })

            return candidates

        # --- MODO PLANIFICACIÓN MULTI-PASO ---
        candidates = []
        g_lower = self.goal.lower()

        if depth == 0:
            if "camara" in g_lower or "foto" in g_lower or "captura" in g_lower:
                candidates.append({
                    "name": "capture_camera_rust",
                    "tool": "os_camera_capture",
                    "args": {"output_format": "png"},
                    "rationale": "Capturar fotograma con motor nativo Rust nokhwa en microsegundos.",
                    "risk": 0.05,
                    "prior": 0.95
                })
            elif "audio" in g_lower or "volumen" in g_lower or "silencio" in g_lower:
                candidates.append({
                    "name": "adjust_audio_wasapi",
                    "tool": "os_audio_device",
                    "args": {"action": "status"},
                    "rationale": "Consultar estado actual de volumen y mute WASAPI antes de aplicar ajuste.",
                    "risk": 0.05,
                    "prior": 0.95
                })
            elif "excel" in g_lower or "datos" in g_lower or "grafic" in g_lower or "analiz" in g_lower:
                candidates.append({
                    "name": "python_data_pipeline",
                    "tool": "os_python_exec",
                    "args": {"script": "# Pipeline pandas/openpyxl\n"},
                    "rationale": "Procesamiento de datos con pandas/openpyxl en mesa de trabajo.",
                    "risk": 0.1,
                    "prior": 0.92
                })
            else:
                candidates.append({
                    "name": "query_ontology_first",
                    "tool": "query_system_knowledge",
                    "args": {"query": self.goal, "topK": 3},
                    "rationale": "Consultar ontología de capacidades para obtener herramientas exactas.",
                    "risk": 0.0,
                    "prior": 0.90
                })
                candidates.append({
                    "name": "explore_target_context",
                    "tool": "os_explore",
                    "args": {"path": "."},
                    "rationale": "Verificar archivos locales antes de realizar cambios.",
                    "risk": 0.05,
                    "prior": 0.85
                })
        elif depth == 1:
            # Segundo paso: verificación o ejecución consecuente
            candidates.append({
                "name": "verify_result_integrity",
                "tool": "os_analyze_logs",
                "args": {"tail_lines": 20},
                "rationale": "Auditar que el estado del sistema quedó estable tras la primera acción.",
                "risk": 0.0,
                "prior": 0.90
            })
            candidates.append({
                "name": "conclude_with_feedback",
                "tool": "finish_task",
                "args": {"summary": "Objetivo alcanzado con éxito"},
                "rationale": "Sintetizar respuesta final al usuario.",
                "risk": 0.0,
                "prior": 0.85
            })

        return candidates

    def _select(self, node: MCTSNode) -> MCTSNode:
        """Fase 1 de MCTS: Selección usando UCB1."""
        current = node
        while current.children:
            # Si hay hijos pendientes de expandir, detenernos para expandir
            candidates = self._get_candidate_actions(current)
            if len(current.children) < len(candidates):
                return current
            # Seleccionar hijo con mayor score UCB1
            current = max(current.children, key=lambda c: c.ucb1_score())
        return current

    def _expand(self, node: MCTSNode) -> Optional[MCTSNode]:
        """Fase 2 de MCTS: Expansión."""
        candidates = self._get_candidate_actions(node)
        existing_tools = {c.action["tool"] for c in node.children}

        for cand in candidates:
            if cand["tool"] not in existing_tools:
                child = MCTSNode(
                    action=cand,
                    parent=node,
                    prior=cand.get("prior", 1.0),
                    depth=node.depth + 1
                )
                node.children.append(child)
                return child
        return node if not node.children else node.children[0]

    def _simulate(self, node: MCTSNode) -> float:
        """Fase 3 de MCTS: Simulación Heurística (Evaluación rápida de recompensa)."""
        act = node.action
        reward = 0.5  # Base

        # Recompensa por alineación con la meta
        g_words = set(self.goal.lower().split())
        act_words = set((act.get("name", "") + " " + act.get("rationale", "")).lower().split())
        overlap = len(g_words.intersection(act_words))
        reward += min(0.35, overlap * 0.08)

        # Penalización por Blast Radius / Riesgo destructivo sin confirmar
        risk = act.get("risk", 0.0)
        reward -= (risk * 0.3)

        # Bonus por auto-verificación y bajo riesgo
        if act.get("tool") in ["query_system_knowledge", "os_explore", "os_find_files", "os_camera_capture"]:
            reward += 0.15

        # Penalización severa por repetir la misma herramienta fallida
        failed_tool = self.error_context.get("failed_tool")
        if failed_tool and act.get("tool") == failed_tool:
            reward -= 0.6

        return max(0.0, min(1.0, reward))

    def _backpropagate(self, node: MCTSNode, reward: float):
        """Fase 4 de MCTS: Retropropagación de visitas y recompensas."""
        curr = node
        while curr:
            curr.visits += 1
            curr.value += reward
            curr = curr.parent

    def run(self) -> Dict[str, Any]:
        start_time = time.perf_counter()

        for _ in range(self.iterations):
            leaf = self._select(self.root)
            expanded = self._expand(leaf)
            target = expanded if expanded else leaf
            reward = self._simulate(target)
            self._backpropagate(target, reward)

        duration_ms = (time.perf_counter() - start_time) * 1000

        # Encontrar la mejor trayectoria
        trajectory = []
        curr = self.root
        step_idx = 1

        while curr.children:
            best_child = max(curr.children, key=lambda c: c.visits)
            trajectory.append({
                "step": step_idx,
                "action_name": best_child.action.get("name", ""),
                "tool": best_child.action.get("tool", ""),
                "args": best_child.action.get("args", {}),
                "rationale": best_child.action.get("rationale", ""),
                "visits": best_child.visits,
                "confidence": round(best_child.q_value, 3)
            })
            step_idx += 1
            curr = best_child

        # Resumen de recuperación
        recovery_advice = ""
        if self.error_context.get("failed_tool"):
            if trajectory:
                top = trajectory[0]
                recovery_advice = (
                    f"Ruta óptima tras fallo de '{self.error_context.get('failed_tool')}': "
                    f"Ejecutar '{top['tool']}' ({top['rationale']}) con confianza {top['confidence']}."
                )

        return {
            "status": "success",
            "goal": self.goal,
            "total_simulations": self.iterations,
            "duration_ms": round(duration_ms, 2),
            "optimal_trajectory": trajectory,
            "recovery_advice": recovery_advice
        }

def main():
    parser = argparse.ArgumentParser(description="OzyAssist MCTS Cognitive Planner")
    parser.add_argument("--goal", type=str, required=True, help="Objetivo o tarea a planificar")
    parser.add_argument("--context", type=str, default="", help="Contexto adicional")
    parser.add_argument("--failed_tool", type=str, default="", help="Herramienta que falló si hay error")
    parser.add_argument("--error_msg", type=str, default="", help="Mensaje de error observado")
    parser.add_argument("--iterations", type=int, default=60, help="Número de iteraciones MCTS")
    args = parser.parse_args()

    error_ctx = {}
    if args.failed_tool or args.error_msg:
        error_ctx = {
            "failed_tool": args.failed_tool,
            "error_message": args.error_msg
        }

    planner = MCTSPlanner(
        goal=args.goal,
        context_data=args.context,
        error_context=error_ctx,
        iterations=args.iterations
    )
    result = planner.run()
    print(json.dumps(result, ensure_ascii=False, indent=2))

if __name__ == "__main__":
    main()
