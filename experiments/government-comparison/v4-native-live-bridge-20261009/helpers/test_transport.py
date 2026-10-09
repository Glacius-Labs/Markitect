"""Pure offline transport fixtures; no Main/model/HTTP/CLI/server process."""
import hashlib,json,tempfile,threading,time,unittest
from pathlib import Path
import live_bridge as b

class Transport(unittest.TestCase):
    def request(self):return json.dumps({'runId':'a'*32,'nonce':'b'*32,'inputDigest':'sha256:fixture',
                                       'request':{'role':'executor'}}).encode()
    def policy(self):return {'roles':['executor'],'maxRequests':1,'bridgeSeconds':2,
                             'notAfterEpoch':time.time()+2,'maxResponseBytes':1024}
    def delivery(self,root,raw,valid=True):
        call=root/('a'*32)
        while not (call/'ready.json').exists():time.sleep(.005)
        response=b'  {"fixture":"raw bytes unchanged"}\n'
        (call/'response.raw.json').write_bytes(response)
        terminal=b'{"fixture":"completed"}'
        (call/'native-terminal.json').write_bytes(terminal)
        b.publish(call/'delivery.json',json.dumps({'invocationSha256':b.digest(raw),
            'nativeActorTerminal':valid,'nativeActorId':'fixture-not-live',
            'terminalReceiptSha256':b.digest(terminal),'responseSha256':b.digest(response)}).encode())
        return response
    def test_exact_bytes_forwarded_only_after_terminal(self):
        with tempfile.TemporaryDirectory() as d:
            root=Path(d);raw=self.request();thread=threading.Thread(target=self.delivery,args=(root,raw));thread.start()
            result=b.transfer(raw,root,self.policy());thread.join()
            self.assertEqual(result,b'  {"fixture":"raw bytes unchanged"}\n')
            self.assertEqual((root/('a'*32)/'invocation.raw.json').read_bytes(),raw)
    def test_no_terminal_refuses(self):
        with tempfile.TemporaryDirectory() as d:
            root=Path(d);raw=self.request();thread=threading.Thread(target=self.delivery,args=(root,raw,False));thread.start()
            with self.assertRaises(RuntimeError):b.transfer(raw,root,self.policy())
            thread.join()
    def test_missing_delivery_timeout(self):
        with tempfile.TemporaryDirectory() as d:
            policy=self.policy();policy['bridgeSeconds']=.02
            with self.assertRaises(TimeoutError):b.transfer(self.request(),Path(d),policy)
    def test_failed_first_request_does_not_refill(self):
        with tempfile.TemporaryDirectory() as d:
            policy=self.policy();policy['bridgeSeconds']=.02
            with self.assertRaises(TimeoutError):b.transfer(self.request(),Path(d),policy)
            with self.assertRaisesRegex(RuntimeError,'quota'):b.transfer(self.request(),Path(d),policy)
    def test_role_and_path_refuse(self):
        with tempfile.TemporaryDirectory() as d:
            raw=json.loads(self.request());raw['runId']='../escape'
            with self.assertRaises(RuntimeError):b.transfer(json.dumps(raw).encode(),Path(d),self.policy())
            raw['runId']='c'*32;raw['request']['role']='verifier'
            with self.assertRaises(RuntimeError):b.transfer(json.dumps(raw).encode(),Path(d),self.policy())
    def test_publication_never_overwrites(self):
        with tempfile.TemporaryDirectory() as d:
            p=Path(d)/'artifact';b.publish(p,b'first')
            with self.assertRaises(RuntimeError):b.publish(p,b'second')
            self.assertEqual(p.read_bytes(),b'first')

if __name__=='__main__':unittest.main()
