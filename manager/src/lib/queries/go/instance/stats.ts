import { useQuery } from "@tanstack/react-query";

import { Instance } from "@/types/evolution.types";

import { apiGlobal } from "../../api";

export const useGoInstanceStats = (instance?: Instance | null) => {
  const isGo = instance?.integration === "EVOLUTION_GO";
  const headers = { apikey: instance?.token };

  const contacts = useQuery({
    queryKey: ["go", "contactsCount", instance?.id],
    enabled: !!isGo && !!instance?.token && instance.connectionStatus === "open",
    staleTime: 60_000,
    refetchInterval: 60_000,
    queryFn: async () => {
      const response = await apiGlobal.get<{ data: unknown[] | null }>("/user/contacts", { headers });
      if (response.data.data === null) return 0;
      if (!Array.isArray(response.data.data)) throw new Error("Invalid contacts response");
      return response.data.data.length;
    },
  });

  const messages = useQuery({
    queryKey: ["go", "messagesCount", instance?.id],
    enabled: !!isGo && !!instance?.token,
    staleTime: 10_000,
    refetchInterval: 10_000,
    queryFn: async () => {
      const response = await apiGlobal.get<{ data: number }>("/message/count", { headers });
      if (typeof response.data.data !== "number") throw new Error("Invalid message count response");
      return response.data.data;
    },
  });

  return { contacts, messages };
};
