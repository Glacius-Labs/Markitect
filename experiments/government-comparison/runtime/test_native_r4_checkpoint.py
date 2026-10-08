"""Offline parent stop regressions; synthetic reports are not native evidence."""
import copy
import importlib.util
from pathlib import Path
import unittest

import test_government as translation_fixtures

spec = importlib.util.spec_from_file_location('native_r4_driver', Path(__file__).parents[1] / 'run-native-integration-r4.py')
driver = importlib.util.module_from_spec(spec)
spec.loader.exec_module(driver)


class NativeR4CheckpointTests(unittest.TestCase):
    def setUp(self):
        self.report = translation_fixtures.GovernmentTranslationTests()._report()
        self.report['actors'] = [a for a in self.report['actors'] if a['slotId'] != 'child-writer']
        self.report.update(stage='complete', candidateCommit='b'*40, candidateTree='c'*40,
                           checks=[{'Name':'inventory-overflow','ExitCode':0}])
        self.report['promotion'].update(expectedOld='d'*40, newCommit='b'*40,
            actualActive='b'*40, intentPath='intent.json', completionPath='completion.json')
        self.queue = {'status':'complete','nextStep':'none','inFlightActors':0,
                      'jobs':[{'id':driver.government_integration.TASK_ID,
                               'state':'accepted-scoped','nextStep':'complete'}]}
        self.runtime = {'expectedBase':'d'*40,'checks':[{'name':'inventory-overflow'}],
            'executor':{'slotId':'writer','command':'offline'},
            'verifier':{'slotId':'reviewer','command':'offline'},
            'ressorts':[{'ressort':{'name':'finance'},'runner':{'slotId':'vote-finance','command':'offline'}}]}
        self.process = {'returnCode':0,'stopReason':None}

    def check(self):
        return driver.positive_checkpoint(self.process,self.queue,self.report,self.runtime)

    def test_zero_exit_cannot_admit_incomplete_queue_or_failed_check(self):
        self.queue['status']='incomplete'
        with self.assertRaisesRegex(ValueError,'Queue/process'):
            self.check()
        self.queue['status']='complete'
        self.report['checks'][0]['ExitCode']=1
        with self.assertRaisesRegex(ValueError,'Fresh configured checks'):
            self.check()

    def test_positive_report_requires_vote_identity_review_and_promotion(self):
        self.assertEqual(self.check()['candidateCommit'],'b'*40)
        baseline = copy.deepcopy(self.report)
        for mutation in ('vote','review','promotion'):
            self.report=copy.deepcopy(baseline)
            if mutation=='vote':
                self.report['votes'][0]['evidenceId']='sha256:'+'0'*64
            elif mutation=='review':
                self.report['actors'][1]['result']['Response']['outcome']='incomplete'
            else:
                self.report['promotion']['completionPath']=''
            with self.subTest(mutation=mutation), self.assertRaises(ValueError):
                self.check()


if __name__=='__main__':
    unittest.main()
