#!/usr/bin/env groovy
pipeline {
  agent {
    label 'commonslave'
  }

  stages {

        stage('Prepare') {
          steps {
            step([$class: 'WsCleanup'])
            checkout(scm)
          }
        }

        stage('UnitTest') {
          steps {
              ansiColor('xterm') {
                  sh("make validate test-unit")
              }
          }
        }

        stage('Build') {
          steps {
              ansiColor('xterm') {
                sh("make binary")
              }
          }
        }

	stage('Create new tag') {
          steps {
              ansiColor('xterm') {
                sh("bash set_tag.sh")
              }
          }
        }
  }
}
