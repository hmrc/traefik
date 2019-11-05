#!/usr/bin/env groovy
pipeline {
  agent {
    label 'amislave'
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

  }
}