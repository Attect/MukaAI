import React from "react";

interface ErrorBoundaryProps {
  children: React.ReactNode;
}

interface ErrorBoundaryState {
  hasError: boolean;
  error: Error | null;
}

export default class ErrorBoundary extends React.Component<ErrorBoundaryProps, ErrorBoundaryState> {
  constructor(props: ErrorBoundaryProps) {
    super(props);
    this.state = { hasError: false, error: null };
  }

  static getDerivedStateFromError(error: Error): ErrorBoundaryState {
    return { hasError: true, error };
  }

  componentDidCatch(error: Error, errorInfo: React.ErrorInfo): void {
    console.error("React Error Boundary caught:", error, errorInfo);
  }

  render(): React.ReactNode {
    if (this.state.hasError) {
      return (
        <div className="flex h-screen items-center justify-center bg-gray-900 text-white">
          <div className="text-center max-w-lg">
            <div className="text-2xl font-bold mb-4 text-red-400">渲染错误</div>
            <div className="text-gray-400 text-sm mb-4 whitespace-pre-wrap bg-gray-800 p-4 rounded">
              {this.state.error?.message || "未知错误"}
            </div>
            <button
              onClick={() => this.setState({ hasError: false, error: null })}
              className="bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded"
            >
              重试
            </button>
          </div>
        </div>
      );
    }

    return this.props.children;
  }
}
