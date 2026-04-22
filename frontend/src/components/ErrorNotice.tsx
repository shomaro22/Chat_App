type Props = {
  message: string;
  onRetry?: () => void;
};

export function ErrorNotice({ message, onRetry }: Props) {
  return (
    <div className="error-inline-row">
      <span className="error-text">{message}</span>
      {onRetry && (
        <button type="button" className="pill-button" onClick={onRetry}>
          再試行
        </button>
      )}
    </div>
  );
}
