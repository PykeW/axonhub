/**
 * Formats a raw error message from the backend by extracting request IDs
 * and cleaning up the message.
 */
export function formatErrorMessage(error: string): {
  message: string;
  requestIDs: string[];
} {
  if (!error) return { message: '', requestIDs: [] };

  const requestIDs: string[] = [];
  const requestIdRegex = /\(request id: ([^)]+)\)/g;

  let match;
  let cleanedMessage = error;

  while ((match = requestIdRegex.exec(error)) !== null) {
    requestIDs.push(match[1]);
  }

  // Remove the (request id: ...) parts from the message
  cleanedMessage = error.replace(requestIdRegex, '').trim();

  // Clean up trailing commas or spaces that might remain
  cleanedMessage = cleanedMessage.replace(/,\s*$/, '').trim();

  return {
    message: cleanedMessage,
    requestIDs,
  };
}
