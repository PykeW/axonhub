import { formatErrorMessage } from './error-message-formatter';

/**
 * A component to display formatted error messages and request IDs.
 */
interface ErrorDisplayProps {
  error: string;
  className?: string;
  messageClassName?: string;
  requestIdClassName?: string;
}

export function ErrorDisplay({ 
  error, 
  className = '', 
  messageClassName = 'text-sm font-medium',
  requestIdClassName = 'text-xs text-muted-foreground mt-1 opacity-70'
}: ErrorDisplayProps) {
  const { message, requestIDs } = formatErrorMessage(error);

  if (!error) return null;

  return (
    <div className={className}>
      <div className={messageClassName}>{message}</div>
      {requestIDs.length > 0 && (
        <div className="mt-1 space-y-0.5">
          {requestIDs.map((id, index) => (
            <div key={`${id}-${index}`} className={requestIdClassName}>
              Request ID: {id}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
