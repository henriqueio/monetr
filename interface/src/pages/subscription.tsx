import React from 'react';
import { useNavigate } from 'react-router-dom';
import { LoaderCircle } from 'lucide-react';
import { useSnackbar } from 'notistack';

import CenteredLogo from '@monetr/interface/components/Logo/CenteredLogo';
import MSpan from '@monetr/interface/components/MSpan';
import useMountEffect from '@monetr/interface/hooks/useMountEffect';
import request from '@monetr/interface/util/request';

// SubscriptionPage is just used to redirect the user to the stripe billing portal. Upon mounting, it will make an API
// call to start a billing portal session, and once it gets a response it will redirect the user there.
export default function SubscriptionPage(): JSX.Element {
  const { enqueueSnackbar } = useSnackbar();
  const navigate = useNavigate();

  useMountEffect(() => {
    request().get('/billing/portal')
      .then(result => window.location.assign(result.data.url))
      .catch(error => {
        enqueueSnackbar(error?.response?.data?.error || 'Failed to navigate to billing portal.', {
          variant: 'error',
          disableWindowBlurListener: true,
        });
        navigate('/');
      });
  });

  return (
    <div className='flex items-center justify-center w-full h-full max-h-full'>
      <div className='w-full p-10 xl:w-3/12 lg:w-5/12 md:w-2/3 sm:w-10/12 max-w-screen-sm sm:p-0'>
        <CenteredLogo />
        <div className='w-full pt-2.5 pb-2.5'>
          <MSpan
            size='xl'
            className='w-full text-center'
          >
            Loading the billing portal...
          </MSpan>
        </div>
        <div className='w-full pt-2.5 pb-2.5 flex justify-center'>
          <LoaderCircle className='spin' />
        </div>
      </div>
    </div>
  );
}
