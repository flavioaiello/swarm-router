#!groovy

def project = env.JOB_NAME.split('/').reverse()[1]
def jobshortname = env.JOB_NAME.substring(env.JOB_NAME.lastIndexOf('/') + 1)
def shortname = jobshortname.substring(jobshortname.indexOf('-') + 1)
def dockerImageName = shortname.substring(shortname.indexOf('-') + 1)
def dockerRegistry = 'http://nexus:5000'
def dockerRepository = 'yourrepository'
def dockerCredentialsId = 'docker'

node {
    stage('Checkout') {
        checkout scm
    }

    def dockerImageTag = sh(returnStdout: true, script: 'git describe --all').trim().replaceAll(/(.*\/)?(.+)/,'$2')

    stage('Env') {
        echo '*** Show env variables: ***' + \
             '\n Project: ' + project + \
             '\n Jobshortname: ' + jobshortname + \
             '\n Shortname: ' + shortname + \
             '\n dockerRegistry: ' + dockerRegistry + \
             '\n dockerRepository: ' + dockerRepository + \
             '\n dockerCredentialsId: ' + dockerCredentialsId + \
             '\n dockerImageName: ' + dockerImageName + \
             '\n dockerImageTag: ' + dockerImageTag
    }

    stage('Build & Push') {
        docker.withRegistry(dockerRegistry, dockerCredentialsId) {

            // Set repository and image name
            def image = docker.build dockerRepository + '/' + dockerImageName

            // Push actual tag
            image.push(dockerImageTag)

            // Push latest tag if it's a release
            if ((dockerImageTag ==~ /v(\d+.\d+.\d+)/)) {
                image.push('latest')
            }

            echo '*** Docker image successfully pushed to registry. ***'
        }
    }
}
