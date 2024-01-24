/* eslint-disable max-len */
import React, { useState } from 'react';
import { Navigate } from 'react-router-dom';
import { CheckCircle, EditOutlined, LinkOutlined } from '@mui/icons-material';

import { MBaseButton } from '@monetr/interface/components/MButton';
import MLink from '@monetr/interface/components/MLink';
import MLogo from '@monetr/interface/components/MLogo';
import MSpan from '@monetr/interface/components/MSpan';
import PlaidSetup from '@monetr/interface/components/setup/PlaidSetup';
import TellerSetup from '@monetr/interface/components/setup/TellerSetup';
import { ReactElement } from '@monetr/interface/components/types';
import { useAppConfiguration } from '@monetr/interface/hooks/useAppConfiguration';
import mergeTailwind from '@monetr/interface/util/mergeTailwind';

export interface SetupPageProps {
  alreadyOnboarded?: boolean;
  manualEnabled?: boolean;
}

type Step = 'greeting'|'plaid'|'teller'|'manual'|'loading';

export default function SetupPage(props: SetupPageProps): JSX.Element {
  const [step, setStep] = useState<Step>('greeting');
  const manualPath = props.alreadyOnboarded ? '/link/create/manual' : '/setup/manual';

  switch (step) {
    case 'greeting':
      return <Greeting onContinue={ setStep } manualEnabled={ props.manualEnabled } alreadyOnboarded={ props.alreadyOnboarded } />;
    case 'plaid':
      return <PlaidSetup alreadyOnboarded={ props.alreadyOnboarded } />;
    case 'teller':
      return <TellerSetup />;
    case 'manual':
      return <Navigate to={ manualPath } />;
    case 'loading':
    default:
      return <h1>Something went wrong...</h1>;
  }
}

interface GreetingProps {
  alreadyOnboarded?: boolean;
  manualEnabled: boolean;
  onContinue: (_: Step) => unknown;
}

function Greeting(props: GreetingProps): JSX.Element {
  const config = useAppConfiguration();
  const [active, setActive] = useState<'plaid'|'manual'|null>(null);

  function Banner(): JSX.Element {
    if (!props.alreadyOnboarded) {
      return (
        <div className='flex flex-col justify-center items-center text-center'>
          <MSpan size='2xl' weight='medium'>
            Welcome to monetr!
          </MSpan>
          <MSpan size='lg' color='subtle'>
            Before we get started, please select how you would like to continue.
          </MSpan>
        </div>
      );
    }

    return (
      <div className='flex flex-col justify-center items-center text-center'>
        <MSpan className='text-2xl font-medium'>
          Adding another bank?
        </MSpan>
        <MSpan className='text-lg' color='subtle'>
          Please select what type of bank you want to setup below.
        </MSpan>
      </div>
    );
  }

  function Footer(): JSX.Element {
    if (props.alreadyOnboarded) return null;

    return (
      <div className='flex justify-center gap-1'>
        <MSpan color='subtle' className='text-sm'>Not ready to continue?</MSpan>
        <MLink to='/logout' size='sm'>Logout for now</MLink>
      </div>
    );
  }

  return (
    <div className='w-full h-full flex lg:justify-center items-center gap-4 md:gap-8 flex-col overflow-y-auto py-4'>
      <MLogo className='w-16 h-16 md:w-24 md:h-24' />
      <Banner />
      <div className='flex gap-4 flex-col md:flex-row p-2'>
        <OnboardingTile
          icon={ <LinkOutlined /> }
          name='Connected'
          description='Connect to your bank account automatically using Plaid.'
          active={ active === 'plaid' }
          onClick={ () => setActive('plaid') }
          disabled={ !config?.plaidEnabled }
        />
        <OnboardingTile
          icon={ <EditOutlined /> }
          name='Manual'
          description='Enter transaction and balance data manually.'
          active={ active === 'manual' }
          onClick={ () => setActive('manual') }
          comingSoon={ !props.manualEnabled }
        />
      </div>
      <MBaseButton
        color={ !active ? 'secondary' : 'primary' }
        disabled={ !active }
        onClick={ () => props.onContinue(active) }
      >
        Continue
      </MBaseButton>
      <Footer />
    </div>
  );
}

interface OnboardingTileProps {
  onClick: () => void;
  active: boolean;
  icon: React.ReactElement;
  name: ReactElement;
  description: ReactElement;
  comingSoon?: boolean;
  disabled?: boolean;
}

function OnboardingTile(props: OnboardingTileProps): JSX.Element {
  const nonDisabled = mergeTailwind(
    {
      'dark:border-dark-monetr-brand': props.active,
      'dark:hover:border-dark-monetr-brand-subtle': props.active,
      'border-monetr-brand': props.active,
      'hover:border-monetr-brand-subtle': props.active,
    },
    {
      'dark:border-dark-monetr-border': !props.active,
      'dark:hover:border-dark-monetr-border-string': !props.active,
      'border-monetr-border': !props.active,
      'hover:border-monetr-border-string': !props.active,
    },
    'cursor-pointer',
    'border'
  );
  const disabled = mergeTailwind(
    'cursor-not-allowed',
    'dark:ring-dark-monetr-border-subtle',
    'ring-monetr-border-subtle',
    'ring-1',
    'ring-inset',
    'dark:text-dark-monetr-content-muted',
    'text-monetr-content-muted',
    'opacity-50',
  );

  const disabledState = props.comingSoon || props.disabled;
  const wrapperClasses = mergeTailwind(
    { [nonDisabled]: !disabledState },
    { [disabled]: disabledState },
    'text-center',
    'flex',
    'flex-row',
    'md:flex-col',
    'gap-4',
    'group',
    'md:h-72',
    'md:w-56',
    'items-center',
    'p-2',
    'py-4',
    'md:p-4',
    'relative',
    'rounded-lg',
  );

  function handleClick() {
    if (props.comingSoon) return;

    props.onClick();
  }

  return (
    <a className={ wrapperClasses } onClick={ handleClick }>
      { props.active && <CheckCircle className='absolute dark:text-dark-monetr-brand-subtle top-2 right-2' /> }
      { React.cloneElement(props.icon, { className: 'w-10 h-10 md:w-16 md:h-16 ml-4 md:ml-0 md:mt-2' }) }
      <div className='flex flex-col gap-2 items-center h-full md:mt-4 text-center w-full md:w-auto'>
        <MSpan className='text-lg font-medium'>
          { props.name }
        </MSpan>
        <MSpan color='subtle'>
          { props.description }
        </MSpan>
        { !props.comingSoon && <MSpan>&nbsp;</MSpan>}
        { props.comingSoon &&
          <MSpan className='md:mt-5 font-medium'>
            Coming Soon
          </MSpan>
        }
        { props.disabled &&
          <MSpan className='md:mt-5 font-medium'>
            Unavailable
          </MSpan>
        }
      </div>
    </a>
  );
}

