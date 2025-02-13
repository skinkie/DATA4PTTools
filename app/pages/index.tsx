import EmailIcon from '@mui/icons-material/Email'
import GitHubIcon from '@mui/icons-material/GitHub'
import { Alert, Button, Stack, Typography } from '@mui/material'
import { NextPage } from 'next'
import { useRouter } from 'next/router'
import React from 'react'
import ErrorAlert from '../components/ErrorAlert'
import FullscreenLoader from '../components/FullscreenLoader'
import MainContent from '../components/MainContent'
import useApiClient from '../hooks/useApiClient'

const Home: NextPage = () => {
  const [errorMessage, setErrorMessage] = React.useState<string>('')
  const [errorOpen, setErrorOpen] = React.useState<boolean>(false)
  const [loading, setLoading] = React.useState<boolean>(false)
  const router = useRouter()
  const apiClient = useApiClient()

  const handleClick = (): void => {
    setLoading(true)

    apiClient.createSession()
      .then(async (session) => {
        await router.push('/jobs/' + session.id)
      })
      .catch(err => setErrorMessage(err.message))
      .finally(() => setLoading(false))
  }

  return (
    <React.Fragment>
      <ErrorAlert
        open={errorOpen}
        message={errorMessage}
        onClose={() => setErrorOpen(false)}
      />
      <MainContent>
        <Stack
          spacing={{ xs: 2, md: 4 }}
          justifyContent="center"
        >
          <Stack spacing={4}>
            <Stack spacing={2}>
              <Typography variant="h3">Data4PT</Typography>
              <Typography>
  The DATA4PT project aims to advance data-sharing practices in the public transport sector by supporting the development of data exchange standards and models, to fulfil the needs of multimodal travel information service providers.
              </Typography>
              <Typography gutterBottom>By supporting EU Member States in deploying a set of harmonised European public data standards (Transmodel, NeTEx and SIRI), DATA4PT wants to enable union-wide multimodal travel information services and contribute to a seamless door-to-door travel ecosystem across Europe that covers all mobility services.</Typography>
            </Stack>
            <Stack spacing={2}>
              <Typography variant="h3">Validation tool</Typography>
              <Typography>
                Key activity of DATA4PT project is the development of validation tools for NeTEx and SIRI datasets. As NeTEx and SIRI are the EU standardised formats for public transport data in National Access Points (NAPs), the purpose of validation is to ensure a certain level of quality of the published data. The quality dimension is aligned with the overall objective of the project to enable the implementation of ITS Directive Delegated Regulation EU 2017/1926 and therefore the interoperable exchange of travel and traffic data across Europe.
              </Typography>
              <Typography>
                If you have feedback, questions or bug reports please do not hesitate to send them our way through <a href="https://github.com/ITxPT/DATA4PTTools" target="_blank" rel="noreferrer"><GitHubIcon sx={{ fontSize: '12px' }} /> GitHub</a> or <a target="_blank" href="mailto:anastasia.founta@itxpt.org;jesper.j.tornros@concreteit.se;petter.kvarnfors@concreteit.se" rel="noreferrer"><EmailIcon sx={{ fontSize: '12px' }} /> Email</a>.
              </Typography>
            </Stack>
          </Stack>
          <Alert severity="info">
            Using the online version may apply limitations. For regular use, download and install the tool for free from <a href="https://github.com/ITxPT/DATA4PTTools" target="_blank" rel="noreferrer">GitHub</a> or <a href="https://hub.docker.com/r/lekojson/greenlight" target="_blank" rel="noreferrer">Docker</a>
          </Alert>
          <Stack alignItems="center">
            <Button variant="contained" onClick={handleClick}>Start validating</Button>
          </Stack>
        </Stack>

        <FullscreenLoader open={loading} />
      </MainContent>
    </React.Fragment>
  )
}

export default Home
