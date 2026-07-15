import { Component, type ReactNode } from "react";

interface Props {
  children: ReactNode;
}

interface State {
  hasError: boolean;
  error?: Error;
}

export default class ErrorBoundary extends Component<Props, State> {
  constructor(props: Props) {
    super(props);
    this.state = { hasError: false };
  }

  static getDerivedStateFromError(error: Error): State {
    return { hasError: true, error };
  }

  render() {
    if (this.state.hasError) {
      return (
        <div className="flex-1 flex items-center justify-center bg-surface-deep">
          <div className="text-center max-w-md p-8">
            <span className="material-symbols-outlined text-[48px] text-text-muted mb-4">error</span>
            <h2 className="text-headline-md font-headline-md text-on-surface mb-2">Algo salió mal</h2>
            <p className="text-text-muted text-body-md mb-4">
              {this.state.error?.message || "Ocurrió un error inesperado."}
            </p>
            <button
              className="px-4 py-2 bg-primary-container text-on-primary rounded-lg hover:bg-primary-fixed-dim transition-colors"
              onClick={() => window.location.reload()}
            >
              Recargar página
            </button>
          </div>
        </div>
      );
    }
    return this.props.children;
  }
}
