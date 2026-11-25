/** Runs a shop call, handling busy state, errors and the request log in the shell. */
export type Runner = <T>(label: string, f: () => Promise<T>) => Promise<T | undefined>;
