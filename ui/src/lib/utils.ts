import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";
import useSWR from "swr";

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

export function useSWRFetch<T>(url: string, setWorking?: (v: boolean) => void) {
  return useSWR<T>(url, async (): Promise<T> => {
    if (setWorking) {
      setWorking(true);
    }

    const resp = await fetch(url);

    if (!resp.ok) {
      throw new Error(
        `Failed to fetch: ${resp.statusText}: ${await resp.text()}`,
      );
    }
    const json = await resp.json();
    if (setWorking) {
      setWorking(false);
    }
    return json;
  });
}
