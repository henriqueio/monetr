import * as Sentry from '@sentry/react';
import { useMutation, useQuery, useQueryClient, UseQueryResult } from '@tanstack/react-query';

import User from '@monetr/interface/models/User';
import parseDate from '@monetr/interface/util/parseDate';
import request from '@monetr/interface/util/request';

export interface AuthenticationWrapper {
  user: User;
  defaultCurrency: string;
  mfaPending: boolean;
  isSetup: boolean;
  isActive: boolean;
  isTrialing: boolean;
  activeUntil: Date | null;
  trialingUntil: Date | null;
  hasSubscription: boolean;
}

export type AuthenticationResult =
  { result: AuthenticationWrapper }
  & UseQueryResult<Partial<AuthenticationWrapper>, unknown>;

export function useAuthenticationSink(): AuthenticationResult {
  const result = useQuery<Partial<AuthenticationWrapper>>(['/users/me'], {
    onSuccess: data => {
      if (data?.user?.accountId) {
        Sentry.setUser({
          id: data.user.accountId,
          username: `account:${data.user.accountId}`,
        });
      }
    },
    refetchOnWindowFocus: true, // Might want to change this to 'always' at some point?
  });
  return {
    ...result,
    result: {
      user: result?.data?.user && new User(result?.data?.user),
      defaultCurrency: result?.data?.defaultCurrency,
      mfaPending: Boolean(result?.data?.mfaPending),
      isSetup: Boolean(result?.data?.isSetup),
      isActive: Boolean(result?.data?.isActive),
      isTrialing: Boolean(result?.data?.isTrialing),
      activeUntil: parseDate(result?.data?.activeUntil),
      trialingUntil: parseDate(result?.data?.trialingUntil),
      hasSubscription: Boolean(result?.data?.hasSubscription),
    },
  };
}

export function useAuthentication(): User | null {
  const { result: { user } } = useAuthenticationSink();
  return user;
}

export interface AfterCheckoutResult {
  message: string | null;
  nextUrl: string;
  isActive: boolean;
}

// useAfterCheckout is a hook that provides a function where the caller can give a Stripe checkout session ID which is
// used to refresh the state of the currently authenticated user's subscription. This is intended to be used after a
// user has been redirected back to the application from Stripe to see if their subscription is now/still active.
// The function yielded by this hook will return the result of that "after checkout" data. But will also mutate the
// `isActive` variable from `useAuthentication` to properly represent the new subscription status.
export function useAfterCheckout(): (_checkoutSessionId: string) => Promise<AfterCheckoutResult> {
  const queryClient = useQueryClient();

  async function queryCheckoutSession(checkoutSessionId: string): Promise<AfterCheckoutResult> {
    return request()
      .get<AfterCheckoutResult>(`/billing/checkout/${checkoutSessionId}`)
      .then(result => result.data);
  }

  const mutation = useMutation(
    queryCheckoutSession,
    {
      onSuccess: () => Promise.all([
        queryClient.invalidateQueries(['/users/me']),
      ]),
    },
  );

  return mutation.mutateAsync;
}
